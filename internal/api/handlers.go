package api

import (
	"net/http"
	"strconv"

	"datacenter/internal/auth"
	"datacenter/internal/models"
	"datacenter/internal/scraper"
	"datacenter/internal/storage"
	"datacenter/pkg/jql"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Handler struct {
	storage     storage.Storage
	rbacStorage storage.RBACStorage
	scraper     scraper.Scraper
}

func NewHandler(businessStorage storage.Storage, rbacStorage storage.RBACStorage, scraper scraper.Scraper) *Handler {
	return &Handler{
		storage:     businessStorage,
		rbacStorage: rbacStorage,
		scraper:     scraper,
	}
}

func (h *Handler) RegisterRoutes(router *gin.Engine, jwtService auth.JWTService) {
	public := router.Group("/api")
	{
		public.Use(func(c *gin.Context) {
			c.Set("jwtService", jwtService)
			c.Next()
		})
		public.POST("/auth/login", h.Login)
	}

	protected := router.Group("/api")
	protected.Use(auth.AuthMiddleware(jwtService))
	{
		users := protected.Group("/users")
		{
			users.POST("", h.CreateUser)
			users.GET("", h.GetUsers)
			users.GET("/:id", h.GetUserByID)
			users.PUT("/:id", h.UpdateUser)
			users.DELETE("/:id", h.DeleteUser)
			users.POST("/:id/roles", h.AssignRoleToUser)
			users.DELETE("/:id/roles/:roleId", h.RemoveRoleFromUser)
			users.GET("/:id/roles", h.GetUserRoles)
		}

		permissions := protected.Group("/permissions")
		{
			permissions.POST("", h.CreatePermission)
			permissions.GET("", h.GetPermissions)
			permissions.GET("/:id", h.GetPermissionByID)
			permissions.PUT("/:id", h.UpdatePermission)
			permissions.DELETE("/:id", h.DeletePermission)
		}

		roles := protected.Group("/roles")
		{
			roles.POST("", h.CreateRole)
			roles.GET("", h.GetRoles)
			roles.GET("/:id", h.GetRoleByID)
			roles.PUT("/:id", h.UpdateRole)
			roles.DELETE("/:id", h.DeleteRole)
			roles.POST("/:id/permissions", h.AssignPermissionToRole)
			roles.DELETE("/:id/permissions/:permissionId", h.RemovePermissionFromRole)
			roles.GET("/:id/permissions", h.GetRolePermissions)
		}

		fields := protected.Group("/fields")
		{
			fields.POST("", h.CreateFieldDefinition)
			fields.GET("/module/:module", h.GetFieldDefinitionsByModule)
			fields.GET("/:id", h.GetFieldDefinitionByID)
			fields.PUT("/:id", h.UpdateFieldDefinition)
			fields.DELETE("/:id", h.DeleteFieldDefinition)
		}

		business := protected.Group("/business")
		{
			business.POST("", h.CreateBusinessData)
			business.GET("/module/:module", h.GetBusinessDataByModule)
			business.GET("/:id", h.GetBusinessDataByID)
			business.PUT("/:id", h.UpdateBusinessData)
			business.DELETE("/:id", h.DeleteBusinessData)
		}

		deleted := protected.Group("/deleted")
		{
			deleted.GET("/module/:module", h.GetDeletedDataByModule)
			deleted.GET("/:id", h.GetDeletedDataByID)
			deleted.POST("/:id/recover", h.RecoverDeletedData)
		}

		scraper := protected.Group("/scraper")
		{
			scraper.POST("/upload", h.UploadScrapeTask)
			scraper.GET("/tasks", h.GetScrapeTasks)
			scraper.GET("/tasks/:id", h.GetScrapeTaskByID)
			scraper.POST("/tasks/:id/retry", h.RetryScrapeTask)
			scraper.DELETE("/tasks/:id", h.DeleteScrapeTask)
		}

		collections := protected.Group("/collections")
		{
			collections.POST("", h.CreateCollection)
			collections.GET("", h.GetCollections)
			collections.GET("/:module", h.GetCollectionByModule)
			collections.PUT("/:module", h.UpdateCollection)
			collections.DELETE("/:module", h.DeleteCollection)
			collections.POST("/:module/indexes", h.CreateCollectionIndex)
			collections.GET("/:module/indexes", h.GetCollectionIndexes)
			collections.DELETE("/:module/indexes/:name", h.DeleteCollectionIndex)
		}
	}
}

func (h *Handler) Login(c *gin.Context) {
	var loginReq struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&loginReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.rbacStorage.GetUserByUsername(loginReq.Username)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
		return
	}

	if err := auth.CheckPassword(loginReq.Password, user.Password); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
		return
	}

	roles, _ := h.rbacStorage.GetUserRoles(user.ID.Hex())
	roleCodes := make([]string, len(roles))
	for i, role := range roles {
		roleCodes[i] = role.Code
	}

	jwtService := c.MustGet("jwtService").(auth.JWTService)
	token, err := jwtService.GenerateToken(user.ID.Hex(), roleCodes, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user": gin.H{
			"id":       user.ID.Hex(),
			"username": user.Username,
			"email":    user.Email,
			"roles":    roleCodes,
		},
	})
}

func (h *Handler) CreateUser(c *gin.Context) {
	var user models.User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	hashedPassword, err := auth.HashPassword(user.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}
	user.Password = hashedPassword

	userID := c.GetString("userID")
	user.CreatedBy = userID
	user.UpdatedBy = userID

	if err := h.rbacStorage.CreateUser(&user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	for _, roleCode := range user.RoleIDs {
		role, err := h.rbacStorage.GetRoleByCode(roleCode)
		if err != nil {
			continue
		}
		h.rbacStorage.AssignRoleToUser(user.ID.Hex(), role.ID.Hex(), userID)
	}

	user.Password = ""

	c.JSON(http.StatusCreated, user)
}

func (h *Handler) GetUsers(c *gin.Context) {
	skip, _ := strconv.ParseInt(c.DefaultQuery("skip", "0"), 10, 64)
	limit, _ := strconv.ParseInt(c.DefaultQuery("limit", "10"), 10, 64)

	users, err := h.rbacStorage.GetUsers(skip, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	for i := range users {
		users[i].Password = ""
	}

	c.JSON(http.StatusOK, users)
}

func (h *Handler) GetUserByID(c *gin.Context) {
	id := c.Param("id")
	user, err := h.rbacStorage.GetUserByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	roles, _ := h.rbacStorage.GetUserRoles(id)
	roleCodes := make([]string, len(roles))
	for i, role := range roles {
		roleCodes[i] = role.Code
	}
	user.RoleIDs = roleCodes

	user.Password = ""

	c.JSON(http.StatusOK, user)
}

func (h *Handler) UpdateUser(c *gin.Context) {
	id := c.Param("id")
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var user models.User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if user.Password != "" {
		hashedPassword, err := auth.HashPassword(user.Password)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
			return
		}
		user.Password = hashedPassword
	}

	user.ID = objectID
	user.UpdatedBy = c.GetString("userID")

	if err := h.rbacStorage.UpdateUser(&user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	user.Password = ""

	c.JSON(http.StatusOK, user)
}

func (h *Handler) DeleteUser(c *gin.Context) {
	id := c.Param("id")
	if err := h.rbacStorage.DeleteUser(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User deleted successfully"})
}

func (h *Handler) AssignRoleToUser(c *gin.Context) {
	userID := c.Param("id")
	var req struct {
		RoleID string `json:"role_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	operatorID := c.GetString("userID")
	if err := h.rbacStorage.AssignRoleToUser(userID, req.RoleID, operatorID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Role assigned successfully"})
}

func (h *Handler) RemoveRoleFromUser(c *gin.Context) {
	userID := c.Param("id")
	roleID := c.Param("roleId")

	if err := h.rbacStorage.RemoveRoleFromUser(userID, roleID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Role removed successfully"})
}

func (h *Handler) GetUserRoles(c *gin.Context) {
	userID := c.Param("id")

	roles, err := h.rbacStorage.GetUserRoles(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, roles)
}

func (h *Handler) CreatePermission(c *gin.Context) {
	var permission models.Permission
	if err := c.ShouldBindJSON(&permission); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetString("userID")
	permission.CreatedBy = userID
	permission.UpdatedBy = userID

	if err := h.rbacStorage.CreatePermission(&permission); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, permission)
}

func (h *Handler) GetPermissions(c *gin.Context) {
	skip, _ := strconv.ParseInt(c.DefaultQuery("skip", "0"), 10, 64)
	limit, _ := strconv.ParseInt(c.DefaultQuery("limit", "10"), 10, 64)

	permissions, err := h.rbacStorage.GetPermissions(skip, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, permissions)
}

func (h *Handler) GetPermissionByID(c *gin.Context) {
	id := c.Param("id")
	permission, err := h.rbacStorage.GetPermissionByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Permission not found"})
		return
	}

	c.JSON(http.StatusOK, permission)
}

func (h *Handler) UpdatePermission(c *gin.Context) {
	id := c.Param("id")
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid permission ID"})
		return
	}

	var permission models.Permission
	if err := c.ShouldBindJSON(&permission); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	permission.ID = objectID
	permission.UpdatedBy = c.GetString("userID")

	if err := h.rbacStorage.UpdatePermission(&permission); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, permission)
}

func (h *Handler) DeletePermission(c *gin.Context) {
	id := c.Param("id")
	if err := h.rbacStorage.DeletePermission(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Permission deleted successfully"})
}

func (h *Handler) CreateRole(c *gin.Context) {
	var role models.Role
	if err := c.ShouldBindJSON(&role); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetString("userID")
	role.CreatedBy = userID
	role.UpdatedBy = userID

	if err := h.rbacStorage.CreateRole(&role); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, role)
}

func (h *Handler) GetRoles(c *gin.Context) {
	skip, _ := strconv.ParseInt(c.DefaultQuery("skip", "0"), 10, 64)
	limit, _ := strconv.ParseInt(c.DefaultQuery("limit", "10"), 10, 64)

	roles, err := h.rbacStorage.GetRoles(skip, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, roles)
}

func (h *Handler) GetRoleByID(c *gin.Context) {
	id := c.Param("id")
	role, err := h.rbacStorage.GetRoleByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Role not found"})
		return
	}

	c.JSON(http.StatusOK, role)
}

func (h *Handler) UpdateRole(c *gin.Context) {
	id := c.Param("id")
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role ID"})
		return
	}

	var role models.Role
	if err := c.ShouldBindJSON(&role); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	role.ID = objectID
	role.UpdatedBy = c.GetString("userID")

	if err := h.rbacStorage.UpdateRole(&role); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, role)
}

func (h *Handler) DeleteRole(c *gin.Context) {
	id := c.Param("id")
	if err := h.rbacStorage.DeleteRole(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Role deleted successfully"})
}

func (h *Handler) AssignPermissionToRole(c *gin.Context) {
	roleID := c.Param("id")
	var req struct {
		PermissionID string `json:"permission_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	operatorID := c.GetString("userID")
	if err := h.rbacStorage.AssignPermissionToRole(roleID, req.PermissionID, operatorID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Permission assigned to role successfully"})
}

func (h *Handler) RemovePermissionFromRole(c *gin.Context) {
	roleID := c.Param("id")
	permissionID := c.Param("permissionId")

	if err := h.rbacStorage.RemovePermissionFromRole(roleID, permissionID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Permission removed from role successfully"})
}

func (h *Handler) GetRolePermissions(c *gin.Context) {
	roleID := c.Param("id")

	permissions, err := h.rbacStorage.GetRolePermissions(roleID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, permissions)
}

func (h *Handler) CreateFieldDefinition(c *gin.Context) {
	var field models.FieldDefinition
	if err := c.ShouldBindJSON(&field); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetString("userID")
	field.CreatedBy = userID
	field.UpdatedBy = userID

	if err := h.storage.CreateFieldDefinition(&field); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, field)
}

func (h *Handler) GetFieldDefinitionsByModule(c *gin.Context) {
	module := c.Param("module")
	fields, err := h.storage.GetFieldDefinitionsByModule(module)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, fields)
}

func (h *Handler) GetFieldDefinitionByID(c *gin.Context) {
	id := c.Param("id")
	field, err := h.storage.GetFieldDefinitionByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Field definition not found"})
		return
	}

	c.JSON(http.StatusOK, field)
}

func (h *Handler) UpdateFieldDefinition(c *gin.Context) {
	id := c.Param("id")
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid field ID"})
		return
	}

	var field models.FieldDefinition
	if err := c.ShouldBindJSON(&field); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	field.ID = objectID
	field.UpdatedBy = c.GetString("userID")

	if err := h.storage.UpdateFieldDefinition(&field); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, field)
}

func (h *Handler) DeleteFieldDefinition(c *gin.Context) {
	id := c.Param("id")
	if err := h.storage.DeleteFieldDefinition(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Field definition deleted successfully"})
}

func (h *Handler) CreateBusinessData(c *gin.Context) {
	var data models.BusinessData
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetString("userID")
	data.CreatedBy = userID
	data.UpdatedBy = userID

	if err := h.storage.CreateBusinessData(&data); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, data)
}

func (h *Handler) GetBusinessDataByModule(c *gin.Context) {
	module := c.Param("module")
	skip, _ := strconv.ParseInt(c.DefaultQuery("skip", "0"), 10, 64)
	limit, _ := strconv.ParseInt(c.DefaultQuery("limit", "10"), 10, 64)

	jqlQuery := c.Query("jql")
	filter := bson.M{}
	if jqlQuery != "" {
		var err error
		filter, err = jql.ParseQuery(jqlQuery)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JQL query: " + err.Error()})
			return
		}
	}

	dataList, err := h.storage.GetBusinessDataByModule(module, filter, skip, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, dataList)
}

func (h *Handler) GetBusinessDataByID(c *gin.Context) {
	id := c.Param("id")
	data, err := h.storage.GetBusinessDataByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Business data not found"})
		return
	}

	c.JSON(http.StatusOK, data)
}

func (h *Handler) UpdateBusinessData(c *gin.Context) {
	id := c.Param("id")
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid data ID"})
		return
	}

	var data models.BusinessData
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	data.ID = objectID
	data.UpdatedBy = c.GetString("userID")

	if err := h.storage.UpdateBusinessData(&data); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, data)
}

func (h *Handler) DeleteBusinessData(c *gin.Context) {
	id := c.Param("id")
	userID := c.GetString("userID")

	if err := h.storage.DeleteBusinessData(id, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Business data deleted successfully"})
}

func (h *Handler) GetDeletedDataByModule(c *gin.Context) {
	module := c.Param("module")
	skip, _ := strconv.ParseInt(c.DefaultQuery("skip", "0"), 10, 64)
	limit, _ := strconv.ParseInt(c.DefaultQuery("limit", "10"), 10, 64)

	dataList, err := h.storage.GetDeletedDataByModule(module, skip, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, dataList)
}

func (h *Handler) GetDeletedDataByID(c *gin.Context) {
	id := c.Param("id")
	data, err := h.storage.GetDeletedDataByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Deleted data not found"})
		return
	}

	c.JSON(http.StatusOK, data)
}

func (h *Handler) RecoverDeletedData(c *gin.Context) {
	id := c.Param("id")
	userID := c.GetString("userID")

	if err := h.storage.RecoverDeletedData(id, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Data recovered successfully"})
}

// 刮削任务相关处理函数

func (h *Handler) UploadScrapeTask(c *gin.Context) {
	var req struct {
		DataPath    string `json:"data_path" binding:"required"`
		ScraperPath string `json:"scraper_path" binding:"required"`
		Module      string `json:"module" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetString("userID")

	// 创建刮削任务
	task := &models.ScrapeTask{
		Module:      req.Module,
		DataPath:    req.DataPath,
		ScraperPath: req.ScraperPath,
		Status:      models.ScrapeTaskStatusScraping,
		BaseModel: models.BaseModel{
			CreatedBy: userID,
			UpdatedBy: userID,
		},
	}

	// 提交任务到刮削系统
	err := h.scraper.SubmitTask(task)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Scrape task submitted successfully",
		"task_id": task.ID.Hex(),
	})
}

func (h *Handler) GetScrapeTasks(c *gin.Context) {
	module := c.Query("module")
	status := c.Query("status")
	skip, _ := strconv.ParseInt(c.DefaultQuery("skip", "0"), 10, 64)
	limit, _ := strconv.ParseInt(c.DefaultQuery("limit", "10"), 10, 64)

	tasks, err := h.storage.GetScrapeTasksByModule(module, status, skip, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, tasks)
}

func (h *Handler) GetScrapeTaskByID(c *gin.Context) {
	id := c.Param("id")
	task, err := h.storage.GetScrapeTaskByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Scrape task not found"})
		return
	}

	c.JSON(http.StatusOK, task)
}

func (h *Handler) RetryScrapeTask(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		ScraperPath string `json:"scraper_path" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 获取原任务
	task, err := h.storage.GetScrapeTaskByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Scrape task not found"})
		return
	}

	// 创建新任务
	userID := c.GetString("userID")
	newTask := &models.ScrapeTask{
		Module:      task.Module,
		DataPath:    task.DataPath,
		ScraperPath: req.ScraperPath,
		Status:      models.ScrapeTaskStatusScraping,
		BaseModel: models.BaseModel{
			CreatedBy: userID,
			UpdatedBy: userID,
		},
	}

	// 提交新任务
	err = h.scraper.SubmitTask(newTask)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Scrape task retried successfully",
		"task_id": newTask.ID.Hex(),
	})
}

func (h *Handler) DeleteScrapeTask(c *gin.Context) {
	id := c.Param("id")
	if err := h.storage.DeleteScrapeTask(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Scrape task deleted successfully"})
}

// 集合管理相关处理函数

func (h *Handler) CreateCollection(c *gin.Context) {
	var collection models.Collection
	if err := c.ShouldBindJSON(&collection); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetString("userID")
	collection.CreatedBy = userID
	collection.UpdatedBy = userID

	// 设置集合名称
	if collection.CollectionName == "" {
		collection.CollectionName = collection.Module + "_data"
	}

	if err := h.storage.CreateCollection(&collection); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, collection)
}

func (h *Handler) GetCollections(c *gin.Context) {
	skip, _ := strconv.ParseInt(c.DefaultQuery("skip", "0"), 10, 64)
	limit, _ := strconv.ParseInt(c.DefaultQuery("limit", "10"), 10, 64)

	collections, err := h.storage.GetCollections(skip, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, collections)
}

func (h *Handler) GetCollectionByModule(c *gin.Context) {
	module := c.Param("module")
	collection, err := h.storage.GetCollectionByModule(module)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Collection not found"})
		return
	}

	c.JSON(http.StatusOK, collection)
}

func (h *Handler) UpdateCollection(c *gin.Context) {
	module := c.Param("module")
	var collection models.Collection
	if err := c.ShouldBindJSON(&collection); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	collection.Module = module
	collection.UpdatedBy = c.GetString("userID")

	if err := h.storage.UpdateCollection(&collection); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, collection)
}

func (h *Handler) DeleteCollection(c *gin.Context) {
	module := c.Param("module")
	if err := h.storage.DeleteCollection(module); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Collection deleted successfully"})
}

func (h *Handler) CreateCollectionIndex(c *gin.Context) {
	module := c.Param("module")
	var req struct {
		Keys    bson.M `json:"keys" binding:"required"`
		Options struct {
			Unique     bool   `json:"unique"`
			Background bool   `json:"background"`
			Name       string `json:"name"`
		} `json:"options"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 获取集合信息
	collection, err := h.storage.GetCollectionByModule(module)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Collection not found"})
		return
	}

	// 创建索引选项
	indexOptions := &options.IndexOptions{}
	if req.Options.Unique {
		indexOptions.SetUnique(true)
	}
	if req.Options.Background {
		indexOptions.SetBackground(true)
	}
	if req.Options.Name != "" {
		indexOptions.SetName(req.Options.Name)
	}

	// 创建索引
	err = h.storage.CreateIndex(collection.CollectionName, req.Keys, indexOptions)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Index created successfully"})
}

func (h *Handler) GetCollectionIndexes(c *gin.Context) {
	// 简化实现：返回索引列表
	c.JSON(http.StatusOK, gin.H{
		"message": "Index list not implemented",
	})
}

func (h *Handler) DeleteCollectionIndex(c *gin.Context) {
	// 简化实现：删除索引
	c.JSON(http.StatusOK, gin.H{
		"message": "Index deletion not implemented",
	})
}
