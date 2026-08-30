package router

import (
	"io/fs"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"datacenter/internal/config"
	"datacenter/internal/database"
	"datacenter/internal/errno"
	"datacenter/internal/handler"
	"datacenter/internal/middleware"
	"datacenter/internal/model"
	"datacenter/internal/service"
	"datacenter/internal/store"
	"datacenter/internal/web"
)

// New 构建 Gin 引擎并注册全部路由
func New(cfg *config.Config, db *database.DB) *gin.Engine {
	if cfg.Server.Mode == gin.ReleaseMode {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	// 安全加固：仅信任直连地址（防 X-Forwarded-For 伪造 ClientIP，与登录防爆破联动）；
	// 反向代理部署时按 部署指南.md 配置 TrustedProxies。
	_ = r.SetTrustedProxies(nil)
	r.Use(middleware.RequestLogger(), gin.Recovery(),
		middleware.BodyLimit(4<<20), middleware.SecurityHeaders())

	// 依赖装配
	userStore := store.NewUserStore(db.DB)
	colStore := store.NewCollectionStore(db.DB)
	itemStore := store.NewItemStore(db.DB)
	taskStore := store.NewTaskStore(db.DB)
	relationStore := store.NewRelationStore(db.DB)
	auditStore := store.NewAuditStore(db.DB)

	authService := service.NewAuthService(userStore, auditStore, cfg.JWT)
	userService := service.NewUserService(userStore, auditStore)
	colService := service.NewCollectionService(colStore, userStore, itemStore, taskStore, auditStore)
	itemService := service.NewItemService(itemStore, colStore, taskStore, userStore, cfg.Data.RootDir, auditStore)
	scrapeService := service.NewScrapeService(taskStore, itemStore, colStore, userStore, auditStore)
	relationService := service.NewRelationService(itemStore, colStore, taskStore, userStore, relationStore, auditStore)
	statsService := service.NewStatsService(colStore, itemStore, taskStore, relationStore)
	dqlService := service.NewDQLService(colStore, itemStore, relationStore)
	auditService := service.NewAuditService(auditStore, userStore)

	authHandler := handler.NewAuthHandler(authService)
	userHandler := handler.NewUserHandler(userService)
	colHandler := handler.NewCollectionHandler(colService)
	itemHandler := handler.NewItemHandler(itemService, scrapeService, relationService)
	relationHandler := handler.NewRelationHandler(relationService)
	statsHandler := handler.NewStatsHandler(statsService)
	dqlHandler := handler.NewDQLHandler(dqlService)
	auditHandler := handler.NewAuditHandler(auditService)
	healthHandler := handler.NewHealthHandler(db)

	// 健康检查（不鉴权）
	r.GET("/healthz", healthHandler.Health)

	// API v1
	api := r.Group("/api/v1")
	{
		// 公开接口
		api.POST("/auth/login", authHandler.Login)
	}

	// 需登录
	authed := api.Group("", middleware.Auth(cfg.JWT, userStore))
	{
		authed.GET("/auth/me", authHandler.Me)
		authed.POST("/auth/logout", authHandler.Logout)
		authed.POST("/auth/password", authHandler.ChangePassword)

		// 仪表盘统计
		authed.GET("/stats/overview", statsHandler.Overview)

		// DQL 数据查询（跨集合）
		authed.POST("/dql/query", dqlHandler.Query)
		authed.POST("/dql/aggregate", dqlHandler.Aggregate) // 系统优化 1.2：分组统计

		// 刮削管理：全局任务列表
		authed.GET("/scrape-tasks", itemHandler.GlobalTasks)

		// 集合（集合级权限在 service 中逐集合判定）
		authed.GET("/collections", colHandler.List)
		authed.GET("/collections/:id", colHandler.Get)
		authed.PATCH("/collections/:id", colHandler.UpdateMeta)
		authed.GET("/collections/:id/tags", colHandler.GetTags)
		authed.PUT("/collections/:id/tags", colHandler.PutTags)
		authed.PATCH("/collections/:id/tags", colHandler.PatchTags)
		authed.PUT("/collections/:id/script", colHandler.PutScript)
		authed.PUT("/collections/:id/delete-policy", colHandler.PutDeletePolicy)
		authed.GET("/collections/:id/members", colHandler.ListMembers)
		authed.POST("/collections/:id/members", colHandler.GrantMember)
		authed.DELETE("/collections/:id/members/:userId", colHandler.RemoveMember)

		// 数据项（集合级权限在 service 中逐集合判定）
		authed.POST("/collections/:id/items", itemHandler.Create)
		authed.POST("/collections/:id/items/batch", itemHandler.BatchCreate) // 系统优化 1.1：批量添加
		authed.GET("/collections/:id/items", itemHandler.List)
		authed.GET("/items/search", itemHandler.Search)
		authed.GET("/items/:itemId", itemHandler.Get)
		authed.PATCH("/items/:itemId", itemHandler.Update)
		authed.DELETE("/items/:itemId", itemHandler.Delete)
		authed.POST("/items/:itemId/scrape", itemHandler.Scrape)
		authed.GET("/items/:itemId/scrape-tasks", itemHandler.ListTasks)
		authed.GET("/scrape-tasks/:taskId", itemHandler.GetTask)

		// 关联关系（数据项之间）
		authed.POST("/items/:itemId/relations", relationHandler.Create)
		authed.POST("/items/:itemId/relations/batch", relationHandler.CreateBatch)
		authed.GET("/items/:itemId/relations", relationHandler.List)
		authed.GET("/items/:itemId/tree", relationHandler.Tree)
		authed.GET("/items/:itemId/delete-impact", relationHandler.Impact)
		authed.GET("/items/relation-badges", relationHandler.Badges)
		authed.PATCH("/relations/:relationId", relationHandler.UpdateMeta)
		authed.DELETE("/relations/:relationId", relationHandler.Delete)
	}

	// 公共权限（全局角色 admin 专属）
	admin := authed.Group("", middleware.RequireRole(model.RoleAdmin))
	{
		admin.POST("/users", userHandler.Create)
		admin.GET("/users", userHandler.List)
		admin.PATCH("/users/:id", userHandler.Update)
		admin.DELETE("/users/:id", userHandler.Delete)

		admin.POST("/collections", colHandler.Create)
		admin.DELETE("/collections/:id", colHandler.Delete)
		admin.PUT("/collections/:id/admin", colHandler.AssignAdmin)

		// 审计日志查询（系统优化 3.1，admin 专属）
		admin.GET("/audit-logs", auditHandler.List)
	}

	// 前端静态资源（go:embed）
	staticFS, err := fs.Sub(web.FS, "static")
	if err != nil {
		panic("加载前端静态资源失败: " + err.Error())
	}
	indexHTML, err := fs.ReadFile(staticFS, "index.html")
	if err != nil {
		panic("读取前端入口失败: " + err.Error())
	}
	r.GET("/static/*filepath", gin.WrapH(http.StripPrefix("/static/", http.FileServer(http.FS(staticFS)))))

	// SPA 兜底：非 API 路径返回 index.html（hash 路由由前端处理）
	r.NoRoute(func(c *gin.Context) {
		p := c.Request.URL.Path
		if strings.HasPrefix(p, "/api/") {
			c.JSON(http.StatusNotFound, gin.H{"code": errno.ErrNotFound.Code, "message": "接口不存在"})
			return
		}
		if p == "/favicon.ico" {
			c.Status(http.StatusNoContent)
			return
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", indexHTML)
	})

	return r
}
