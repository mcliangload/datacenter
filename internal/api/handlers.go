package api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"datacenter/internal/auth"
	"datacenter/internal/models"
	"datacenter/internal/scraper"
	"datacenter/internal/storage"
	"datacenter/pkg/jql"
	"datacenter/pkg/rbac"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Handler struct {
	storage     storage.Storage
	rbacStorage storage.RBACStorage
	scraper     scraper.Scraper
	jwtService  auth.JWTService
	rbacService *rbac.Service
}

func NewHandler(storage storage.Storage, rbacStorage storage.RBACStorage, scraper scraper.Scraper, jwtService auth.JWTService, rbacService *rbac.Service) *Handler {
	return &Handler{
		storage:     storage,
		rbacStorage: rbacStorage,
		scraper:     scraper,
		jwtService:  jwtService,
		rbacService: rbacService,
	}
}

func (h *Handler) RegisterRoutes(r *gin.Engine) {
	r.POST("/api/auth/login", h.Login)
	r.POST("/api/auth/register", h.Register)

	protected := r.Group("/api")
	protected.Use(h.AuthMiddleware())
	{
		users := protected.Group("/users")
		users.Use(h.PermissionMiddleware(rbac.PermissionUserRead))
		{
			users.GET("", h.GetUsers)
			users.GET("/:id", h.GetUserByID)
			users.POST("", h.CreateUser).Use(h.PermissionMiddleware(rbac.PermissionUserWrite))
			users.PUT("/:id", h.UpdateUser).Use(h.PermissionMiddleware(rbac.PermissionUserWrite))
			users.DELETE("/:id", h.DeleteUser).Use(h.PermissionMiddleware(rbac.PermissionUserWrite))
			users.POST("/:id/roles", h.AssignRoleToUser).Use(h.PermissionMiddleware(rbac.PermissionUserWrite))
			users.DELETE("/:id/roles/:roleId", h.RemoveRoleFromUser).Use(h.PermissionMiddleware(rbac.PermissionUserWrite))
			users.GET("/:id/roles", h.GetUserRoles)
		}

		permissions := protected.Group("/permissions")
		permissions.Use(h.PermissionMiddleware(rbac.PermissionPermissionRead))
		{
			permissions.GET("", h.GetPermissions)
			permissions.GET("/:id", h.GetPermissionByID)
			permissions.POST("", h.CreatePermission).Use(h.PermissionMiddleware(rbac.PermissionPermissionWrite))
			permissions.PUT("/:id", h.UpdatePermission).Use(h.PermissionMiddleware(rbac.PermissionPermissionWrite))
			permissions.DELETE("/:id", h.DeletePermission).Use(h.PermissionMiddleware(rbac.PermissionPermissionWrite))
		}

		roles := protected.Group("/roles")
		roles.Use(h.PermissionMiddleware(rbac.PermissionRoleRead))
		{
			roles.GET("", h.GetRoles)
			roles.GET("/:id", h.GetRoleByID)
			roles.POST("", h.CreateRole).Use(h.PermissionMiddleware(rbac.PermissionRoleWrite))
			roles.PUT("/:id", h.UpdateRole).Use(h.PermissionMiddleware(rbac.PermissionRoleWrite))
			roles.DELETE("/:id", h.DeleteRole).Use(h.PermissionMiddleware(rbac.PermissionRoleWrite))
			roles.POST("/:id/permissions", h.AssignPermissionToRole).Use(h.PermissionMiddleware(rbac.PermissionRoleWrite))
			roles.DELETE("/:id/permissions/:permissionId", h.RemovePermissionFromRole).Use(h.PermissionMiddleware(rbac.PermissionRoleWrite))
			roles.GET("/:id/permissions", h.GetRolePermissions)
		}

		fields := protected.Group("/fields")
		fields.Use(h.PermissionMiddleware(rbac.PermissionFieldRead))
		{
			fields.GET("/module/:module", h.GetFieldDefinitionsByModule)
			fields.GET("/:id", h.GetFieldDefinitionByID)
			fields.POST("", h.CreateFieldDefinition).Use(h.PermissionMiddleware(rbac.PermissionFieldWrite))
			fields.PUT("/:id", h.UpdateFieldDefinition).Use(h.PermissionMiddleware(rbac.PermissionFieldWrite))
			fields.DELETE("/:id", h.DeleteFieldDefinition).Use(h.PermissionMiddleware(rbac.PermissionFieldWrite))
		}

		business := protected.Group("/business")
		business.Use(h.PermissionMiddleware(rbac.PermissionDataRead))
		{
			business.POST("", h.CreateBusinessData).Use(h.PermissionMiddleware(rbac.PermissionDataWrite))
			business.GET("/module/:module", h.GetBusinessDataByModule)
			business.GET("/module/:module/:id", h.GetBusinessDataByID)
			business.PUT("/module/:module/:id", h.UpdateBusinessData).Use(h.PermissionMiddleware(rbac.PermissionDataWrite))
			business.DELETE("/module/:module/:id", h.DeleteBusinessData).Use(h.PermissionMiddleware(rbac.PermissionDataWrite))
		}

		deleted := protected.Group("/deleted")
		deleted.Use(h.PermissionMiddleware(rbac.PermissionDataRead))
		{
			deleted.GET("/module/:module", h.GetDeletedDataByModule)
			deleted.GET("/:id", h.GetDeletedDataByID)
			deleted.POST("/:id/recover", h.RecoverDeletedData).Use(h.PermissionMiddleware(rbac.PermissionDataWrite))
		}

		scraperGroup := protected.Group("/scraper")
		scraperGroup.Use(h.PermissionMiddleware(rbac.PermissionScrapeRead))
		{
			scraperGroup.POST("/upload", h.SubmitScrapeTask).Use(h.PermissionMiddleware(rbac.PermissionScrapeWrite))
			scraperGroup.GET("/tasks", h.GetScrapeTasks)
			scraperGroup.GET("/tasks/:id", h.GetScrapeTaskByID)
			scraperGroup.POST("/tasks/:id/retry", h.RetryScrapeTask).Use(h.PermissionMiddleware(rbac.PermissionScrapeWrite))
			scraperGroup.DELETE("/tasks/:id", h.DeleteScrapeTask).Use(h.PermissionMiddleware(rbac.PermissionScrapeWrite))
			scraperGroup.POST("/tasks/batch-delete", h.BatchDeleteScrapeTasks).Use(h.PermissionMiddleware(rbac.PermissionScrapeWrite))
		}

		deletedScraper := protected.Group("/deleted-scraper")
		deletedScraper.Use(h.PermissionMiddleware(rbac.PermissionScrapeRead))
		{
			deletedScraper.GET("/module/:module", h.GetDeletedScrapeTasksByModule)
			deletedScraper.GET("/:id", h.GetDeletedScrapeTaskByID)
			deletedScraper.POST("/:id/recover", h.RecoverScrapeTask).Use(h.PermissionMiddleware(rbac.PermissionScrapeWrite))
		}

		collections := protected.Group("/collections")
		collections.Use(h.PermissionMiddleware(rbac.PermissionCollectionRead))
		{
			collections.GET("", h.GetCollections)
			collections.GET("/:module", h.GetCollectionByModule)
			collections.POST("", h.CreateCollection).Use(h.PermissionMiddleware(rbac.PermissionCollectionWrite))
			collections.PUT("/:module", h.UpdateCollection).Use(h.PermissionMiddleware(rbac.PermissionCollectionWrite))
			collections.DELETE("/:module", h.DeleteCollection).Use(h.PermissionMiddleware(rbac.PermissionCollectionWrite))
			collections.POST("/:module/indexes", h.CreateCollectionIndex).Use(h.PermissionMiddleware(rbac.PermissionCollectionWrite))
			collections.GET("/:module/indexes", h.GetCollectionIndexes)
			collections.DELETE("/:module/indexes/:name", h.DeleteCollectionIndex).Use(h.PermissionMiddleware(rbac.PermissionCollectionWrite))
		}
	}
}

func (h *Handler) Login(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.rbacStorage.GetUserByUsername(req.Username)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	if err := auth.CheckPassword(req.Password, user.Password); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	perms, _ := h.rbacService.GetUserPermissions(context.Background(), user.ID.Hex())
	token, err := h.jwtService.GenerateToken(user.ID.Hex(), user.RoleIDs, perms)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
			"email":    user.Email,
			"roles":    user.RoleIDs,
		},
	})
}

func (h *Handler) Register(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
		Email    string `json:"email" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	hashedPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	user := &models.User{
		Username: req.Username,
		Password: hashedPassword,
		Email:    req.Email,
		RoleIDs:  []string{},
	}

	if err := h.rbacStorage.CreateUser(user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":       user.ID,
		"username": user.Username,
		"email":    user.Email,
	})
}

func (h *Handler) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Bearer token required"})
			c.Abort()
			return
		}

		claims, err := h.jwtService.ValidateToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("roles", claims.Roles)

		perms, err := h.rbacService.GetUserPermissions(context.Background(), claims.UserID)
		if err == nil {
			c.Set("permissions", perms)
		}

		c.Next()
	}
}

func (h *Handler) PermissionMiddleware(requiredPerm rbac.Permission) gin.HandlerFunc {
	return func(c *gin.Context) {
		perms, exists := c.Get("permissions")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{"error": "No permissions found"})
			c.Abort()
			return
		}

		permList, ok := perms.([]string)
		if !ok {
			c.JSON(http.StatusForbidden, gin.H{"error": "Invalid permissions format"})
			c.Abort()
			return
		}

		required := string(requiredPerm)
		for _, p := range permList {
			if p == required || (strings.HasSuffix(p, ":*") && strings.HasPrefix(required, strings.TrimSuffix(p, "*"))) {
				c.Next()
				return
			}
		}

		c.JSON(http.StatusForbidden, gin.H{"error": "Permission denied: " + required})
		c.Abort()
	}
}

func (h *Handler) GetUsers(c *gin.Context) {
	page, _ := strconv.ParseInt(c.DefaultQuery("page", "1"), 10, 64)
	pageSize, _ := strconv.ParseInt(c.DefaultQuery("pageSize", "10"), 10, 64)
	skip := (page - 1) * pageSize

	users, err := h.rbacStorage.GetUsers(skip, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	total, err := h.rbacStorage.GetUsersCount()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	for i := range users {
		users[i].Password = ""
	}

	c.JSON(http.StatusOK, gin.H{
		"data":     users,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

func (h *Handler) GetUserByID(c *gin.Context) {
	id := c.Param("id")
	user, err := h.rbacStorage.GetUserByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	user.Password = ""
	c.JSON(http.StatusOK, user)
}

func (h *Handler) CreateUser(c *gin.Context) {
	var req struct {
		Username string   `json:"username" binding:"required"`
		Password string   `json:"password" binding:"required"`
		Email    string   `json:"email" binding:"required"`
		RoleIDs  []string `json:"role_ids"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	hashedPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	user := &models.User{
		Username: req.Username,
		Password: hashedPassword,
		Email:    req.Email,
		RoleIDs:  req.RoleIDs,
	}

	if err := h.rbacStorage.CreateUser(user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	user.Password = ""
	c.JSON(http.StatusCreated, user)
}

func (h *Handler) UpdateUser(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Email    string   `json:"email"`
		Password string   `json:"password"`
		RoleIDs  []string `json:"role_ids"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.rbacStorage.GetUserByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	if req.Email != "" {
		user.Email = req.Email
	}
	if req.Password != "" {
		hashedPassword, err := auth.HashPassword(req.Password)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
			return
		}
		user.Password = hashedPassword
	}
	if req.RoleIDs != nil {
		user.RoleIDs = req.RoleIDs
	}

	if err := h.rbacStorage.UpdateUser(user); err != nil {
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

	user, err := h.rbacStorage.GetUserByID(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	for _, roleID := range user.RoleIDs {
		if roleID == req.RoleID {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Role already assigned"})
			return
		}
	}

	user.RoleIDs = append(user.RoleIDs, req.RoleID)
	if err := h.rbacStorage.UpdateUser(user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, user)
}

func (h *Handler) RemoveRoleFromUser(c *gin.Context) {
	userID := c.Param("id")
	roleID := c.Param("roleId")

	user, err := h.rbacStorage.GetUserByID(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	newRoleIDs := []string{}
	for _, id := range user.RoleIDs {
		if id != roleID {
			newRoleIDs = append(newRoleIDs, id)
		}
	}

	user.RoleIDs = newRoleIDs
	if err := h.rbacStorage.UpdateUser(user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, user)
}

func (h *Handler) GetUserRoles(c *gin.Context) {
	userID := c.Param("id")

	user, err := h.rbacStorage.GetUserByID(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	roles := []models.Role{}
	for _, roleID := range user.RoleIDs {
		role, err := h.rbacStorage.GetRoleByID(roleID)
		if err == nil {
			roles = append(roles, *role)
		}
	}

	c.JSON(http.StatusOK, roles)
}

func (h *Handler) GetPermissions(c *gin.Context) {
	page, _ := strconv.ParseInt(c.DefaultQuery("page", "1"), 10, 64)
	pageSize, _ := strconv.ParseInt(c.DefaultQuery("pageSize", "10"), 10, 64)
	skip := (page - 1) * pageSize

	permissions, err := h.rbacStorage.GetPermissions(skip, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	total, err := h.rbacStorage.GetPermissionsCount()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":     permissions,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
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

func (h *Handler) CreatePermission(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Code        string `json:"code" binding:"required"`
		Description string `json:"description"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	permission := &models.Permission{
		Name:        req.Name,
		Code:        req.Code,
		Description: req.Description,
	}

	if err := h.rbacStorage.CreatePermission(permission); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, permission)
}

func (h *Handler) UpdatePermission(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	permission, err := h.rbacStorage.GetPermissionByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Permission not found"})
		return
	}

	if req.Name != "" {
		permission.Name = req.Name
	}
	if req.Description != "" {
		permission.Description = req.Description
	}

	if err := h.rbacStorage.UpdatePermission(permission); err != nil {
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

func (h *Handler) GetRoles(c *gin.Context) {
	page, _ := strconv.ParseInt(c.DefaultQuery("page", "1"), 10, 64)
	pageSize, _ := strconv.ParseInt(c.DefaultQuery("pageSize", "10"), 10, 64)
	skip := (page - 1) * pageSize

	roles, err := h.rbacStorage.GetRoles(skip, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	total, err := h.rbacStorage.GetRolesCount()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":     roles,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
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

func (h *Handler) CreateRole(c *gin.Context) {
	var req struct {
		Name          string   `json:"name" binding:"required"`
		Code          string   `json:"code" binding:"required"`
		Description   string   `json:"description"`
		PermissionIDs []string `json:"permission_ids"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	role := &models.Role{
		Name:          req.Name,
		Code:          req.Code,
		Description:   req.Description,
		PermissionIDs: req.PermissionIDs,
	}

	if err := h.rbacStorage.CreateRole(role); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, role)
}

func (h *Handler) UpdateRole(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Name          string   `json:"name"`
		Description   string   `json:"description"`
		PermissionIDs []string `json:"permission_ids"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	role, err := h.rbacStorage.GetRoleByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Role not found"})
		return
	}

	if req.Name != "" {
		role.Name = req.Name
	}
	if req.Description != "" {
		role.Description = req.Description
	}
	if req.PermissionIDs != nil {
		role.PermissionIDs = req.PermissionIDs
	}

	if err := h.rbacStorage.UpdateRole(role); err != nil {
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

	role, err := h.rbacStorage.GetRoleByID(roleID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Role not found"})
		return
	}

	for _, permID := range role.PermissionIDs {
		if permID == req.PermissionID {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Permission already assigned"})
			return
		}
	}

	role.PermissionIDs = append(role.PermissionIDs, req.PermissionID)
	if err := h.rbacStorage.UpdateRole(role); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, role)
}

func (h *Handler) RemovePermissionFromRole(c *gin.Context) {
	roleID := c.Param("id")
	permissionID := c.Param("permissionId")

	role, err := h.rbacStorage.GetRoleByID(roleID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Role not found"})
		return
	}

	newPermissionIDs := []string{}
	for _, id := range role.PermissionIDs {
		if id != permissionID {
			newPermissionIDs = append(newPermissionIDs, id)
		}
	}

	role.PermissionIDs = newPermissionIDs
	if err := h.rbacStorage.UpdateRole(role); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, role)
}

func (h *Handler) GetRolePermissions(c *gin.Context) {
	roleID := c.Param("id")

	role, err := h.rbacStorage.GetRoleByID(roleID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Role not found"})
		return
	}

	permissions := []models.Permission{}
	for _, permID := range role.PermissionIDs {
		perm, err := h.rbacStorage.GetPermissionByID(permID)
		if err == nil {
			permissions = append(permissions, *perm)
		}
	}

	c.JSON(http.StatusOK, permissions)
}

func (h *Handler) GetFieldDefinitionsByModule(c *gin.Context) {
	module := c.Param("module")
	fields, err := h.storage.GetFieldDefinitionsByModule(module)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": fields})
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

func (h *Handler) CreateFieldDefinition(c *gin.Context) {
	var req struct {
		Module      string              `json:"module" binding:"required"`
		FieldName   string              `json:"field_name" binding:"required"`
		FieldType   string              `json:"field_type" binding:"required"`
		Description string              `json:"description"`
		Constraints *models.Constraints `json:"constraints"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	field := &models.FieldDefinition{
		Module:      req.Module,
		FieldName:   req.FieldName,
		FieldType:   models.FieldType(req.FieldType),
		Description: req.Description,
		Constraints: *req.Constraints,
	}

	if err := h.storage.CreateFieldDefinition(field); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, field)
}

func (h *Handler) UpdateFieldDefinition(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		FieldName   string              `json:"field_name"`
		FieldType   string              `json:"field_type"`
		Description string              `json:"description"`
		Constraints *models.Constraints `json:"constraints"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	field, err := h.storage.GetFieldDefinitionByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Field definition not found"})
		return
	}

	if req.FieldName != "" {
		field.FieldName = req.FieldName
	}
	if req.FieldType != "" {
		field.FieldType = models.FieldType(req.FieldType)
	}
	if req.Description != "" {
		field.Description = req.Description
	}
	if req.Constraints != nil {
		field.Constraints = *req.Constraints
	}

	if err := h.storage.UpdateFieldDefinition(field); err != nil {
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
	var req struct {
		Module       string                 `json:"module" binding:"required"`
		Data         map[string]interface{} `json:"data"`
		Description  string                 `json:"description"`
		CustomFields map[string]interface{} `json:"custom_fields"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	collection, err := h.storage.GetCollectionByModule(req.Module)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("模块不存在: %s，请先创建模块集合", req.Module)})
		return
	}

	fieldDefs, err := h.storage.GetFieldDefinitionsByModule(req.Module)
	if err == nil && len(fieldDefs) > 0 {
		for _, fieldDef := range fieldDefs {
			value := req.Data[fieldDef.FieldName]
			result := fieldDef.Validate(value)
			if !result.Valid {
				c.JSON(http.StatusBadRequest, gin.H{
					"error":  "字段验证失败",
					"errors": result.Errors,
					"module": req.Module,
					"field":  fieldDef.FieldName,
				})
				return
			}
		}
	}

	userID, _ := c.Get("user_id")
	userIDStr := "unknown"
	if userID != nil {
		userIDStr = userID.(string)
	}

	data := &models.BusinessData{
		Module:       req.Module,
		Description:  req.Description,
		CustomFields: req.CustomFields,
		BaseModel: models.BaseModel{
			CreatedBy: userIDStr,
			CreatedAt: time.Now(),
			UpdatedBy: userIDStr,
			UpdatedAt: time.Now(),
		},
	}

	if req.Data != nil {
		data.CustomFields = req.Data
	}

	ctx := context.Background()
	if err := h.storage.CreateBusinessData(ctx, collection.CollectionName, data); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "数据创建成功",
		"data":    data,
		"module":  req.Module,
	})
}

func (h *Handler) GetBusinessDataByModule(c *gin.Context) {
	module := c.Param("module")
	page, _ := strconv.ParseInt(c.DefaultQuery("page", "1"), 10, 64)
	pageSize, _ := strconv.ParseInt(c.DefaultQuery("pageSize", "10"), 10, 64)
	skip := (page - 1) * pageSize

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

	dataList, err := h.storage.GetBusinessDataByModule(module, filter, skip, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	total, err := h.storage.GetBusinessDataCount(module, filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":     dataList,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

func (h *Handler) GetBusinessDataByID(c *gin.Context) {
	module := c.Param("module")
	id := c.Param("id")
	data, err := h.storage.GetBusinessDataByID(module, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Business data not found"})
		return
	}

	c.JSON(http.StatusOK, data)
}

func (h *Handler) UpdateBusinessData(c *gin.Context) {
	module := c.Param("module")
	id := c.Param("id")

	var req struct {
		Description  string                 `json:"description"`
		Data         map[string]interface{} `json:"data"`
		CustomFields map[string]interface{} `json:"custom_fields"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	data, err := h.storage.GetBusinessDataByID(module, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Business data not found"})
		return
	}

	fieldDefs, err := h.storage.GetFieldDefinitionsByModule(module)
	if err == nil && len(fieldDefs) > 0 && req.Data != nil {
		for _, fieldDef := range fieldDefs {
			value := req.Data[fieldDef.FieldName]
			result := fieldDef.Validate(value)
			if !result.Valid {
				c.JSON(http.StatusBadRequest, gin.H{
					"error":  "字段验证失败",
					"errors": result.Errors,
					"module": module,
					"field":  fieldDef.FieldName,
				})
				return
			}
		}
	}

	userID, _ := c.Get("user_id")
	userIDStr := "unknown"
	if userID != nil {
		userIDStr = userID.(string)
	}

	if req.Description != "" {
		data.Description = req.Description
	}
	if req.Data != nil {
		data.CustomFields = req.Data
	}
	if req.CustomFields != nil {
		data.CustomFields = req.CustomFields
	}
	data.UpdatedBy = userIDStr
	data.UpdatedAt = time.Now()

	if err := h.storage.UpdateBusinessData(data); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, data)
}

func (h *Handler) DeleteBusinessData(c *gin.Context) {
	id := c.Param("id")
	userID, _ := c.Get("user_id")
	userIDStr := "unknown"
	if userID != nil {
		userIDStr = userID.(string)
	}

	if err := h.storage.DeleteBusinessData(id, userIDStr); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Business data deleted successfully"})
}

func (h *Handler) GetDeletedDataByModule(c *gin.Context) {
	module := c.Param("module")
	page, _ := strconv.ParseInt(c.DefaultQuery("page", "1"), 10, 64)
	pageSize, _ := strconv.ParseInt(c.DefaultQuery("pageSize", "10"), 10, 64)
	skip := (page - 1) * pageSize

	dataList, err := h.storage.GetDeletedDataByModule(module, skip, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":     dataList,
		"page":     page,
		"pageSize": pageSize,
	})
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
	userID, _ := c.Get("user_id")
	userIDStr := "unknown"
	if userID != nil {
		userIDStr = userID.(string)
	}

	if err := h.storage.RecoverDeletedData(id, userIDStr); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Data recovered successfully"})
}

func (h *Handler) SubmitScrapeTask(c *gin.Context) {
	var req struct {
		Module      string `json:"module" binding:"required"`
		DataPath    string `json:"data_path" binding:"required"`
		ScraperPath string `json:"scraper_path" binding:"required"`
		Description string `json:"description"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	task := &models.ScrapeTask{
		Module:      req.Module,
		DataPath:    req.DataPath,
		ScraperPath: req.ScraperPath,
		Status:      models.ScrapeTaskStatusPending,
		Description: req.Description,
	}

	userID, _ := c.Get("user_id")
	if userID != nil {
		task.CreatedBy = userID.(string)
	} else {
		task.CreatedBy = "unknown"
	}

	if err := h.scraper.SubmitTask(task); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Scrape task submitted successfully",
		"task_id": task.ID,
	})
}

func (h *Handler) GetScrapeTasks(c *gin.Context) {
	module := c.Query("module")
	status := c.Query("status")
	page, _ := strconv.ParseInt(c.DefaultQuery("page", "1"), 10, 64)
	pageSize, _ := strconv.ParseInt(c.DefaultQuery("pageSize", "10"), 10, 64)
	skip := (page - 1) * pageSize

	tasks, err := h.storage.GetScrapeTasksByModule(module, status, skip, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	total, err := h.storage.GetScrapeTasksCount(module, bson.M{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":     tasks,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
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
		ScraperPath string `json:"scraper_path"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	task, err := h.storage.GetScrapeTaskByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Scrape task not found"})
		return
	}

	if req.ScraperPath != "" {
		task.ScraperPath = req.ScraperPath
	}

	task.Status = models.ScrapeTaskStatusPending
	task.ErrorMessage = ""
	task.StartedAt = nil
	task.CompletedAt = nil

	if err := h.scraper.SubmitTask(task); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Scrape task retry submitted",
		"task_id": task.ID,
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

func (h *Handler) BatchDeleteScrapeTasks(c *gin.Context) {
	var req struct {
		IDs []string `json:"ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	deletedCount := 0
	for _, id := range req.IDs {
		if err := h.storage.DeleteScrapeTask(id); err != nil {
			continue
		}
		deletedCount++
	}

	c.JSON(http.StatusOK, gin.H{"message": "Scrape tasks deleted successfully", "deleted_count": deletedCount})
}

func (h *Handler) GetDeletedScrapeTasksByModule(c *gin.Context) {
	module := c.Param("module")
	// 如果module为"all"，则查询所有模块的删除任务
	if module == "all" {
		module = ""
	}
	page, _ := strconv.ParseInt(c.DefaultQuery("page", "1"), 10, 64)
	pageSize, _ := strconv.ParseInt(c.DefaultQuery("pageSize", "10"), 10, 64)
	skip := (page - 1) * pageSize

	tasks, err := h.storage.GetDeletedScrapeTasks(module, skip, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	total, err := h.storage.GetDeletedScrapeTasksCount(module)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":     tasks,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

func (h *Handler) GetDeletedScrapeTaskByID(c *gin.Context) {
	id := c.Param("id")

	task, err := h.storage.GetDeletedScrapeTaskByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Deleted scrape task not found"})
		return
	}

	c.JSON(http.StatusOK, task)
}

func (h *Handler) RecoverScrapeTask(c *gin.Context) {
	id := c.Param("id")

	if err := h.storage.RecoverScrapeTask(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Scrape task recovered successfully"})
}

func (h *Handler) GetCollections(c *gin.Context) {
	page, _ := strconv.ParseInt(c.DefaultQuery("page", "1"), 10, 64)
	pageSize, _ := strconv.ParseInt(c.DefaultQuery("pageSize", "10"), 10, 64)
	skip := (page - 1) * pageSize

	collections, err := h.storage.GetCollections(skip, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	total, err := h.storage.GetCollectionsCount()
	if err != nil {
		total = int64(len(collections))
	}

	c.JSON(http.StatusOK, gin.H{
		"data":     collections,
		"page":     page,
		"pageSize": pageSize,
		"total":    total,
	})
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

func (h *Handler) CreateCollection(c *gin.Context) {
	var req struct {
		Module        string `json:"module" binding:"required"`
		Description   string `json:"description"`
		DatatypeOwner string `json:"datatype_owner"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	collection := &models.Collection{
		Module:         req.Module,
		Description:    req.Description,
		DatatypeOwner:  req.DatatypeOwner,
		CollectionName: req.Module + "_data",
	}

	userID, _ := c.Get("user_id")
	if userID != nil {
		collection.CreatedBy = userID.(string)
	} else {
		collection.CreatedBy = "unknown"
	}

	if err := h.storage.CreateCollection(collection); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, collection)
}

func (h *Handler) UpdateCollection(c *gin.Context) {
	module := c.Param("module")

	var req struct {
		Description   string `json:"description"`
		DatatypeOwner string `json:"datatype_owner"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	collection, err := h.storage.GetCollectionByModule(module)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Collection not found"})
		return
	}

	if req.Description != "" {
		collection.Description = req.Description
	}
	if req.DatatypeOwner != "" {
		collection.DatatypeOwner = req.DatatypeOwner
	}

	if err := h.storage.UpdateCollection(collection); err != nil {
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
		Keys    map[string]interface{} `json:"keys" binding:"required"`
		Options map[string]interface{} `json:"options"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	collectionName := module + "_data"

	keys := bson.M{}
	for k, v := range req.Keys {
		keys[k] = v
	}

	var opts *options.IndexOptions
	if req.Options != nil {
		opts = options.Index()
		if name, ok := req.Options["name"].(string); ok {
			opts.SetName(name)
		}
		if unique, ok := req.Options["unique"].(bool); ok {
			opts.SetUnique(unique)
		}
		if background, ok := req.Options["background"].(bool); ok {
			opts.SetBackground(background)
		}
	}

	if err := h.storage.CreateIndex(collectionName, keys, opts); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Index created successfully"})
}

func (h *Handler) GetCollectionIndexes(c *gin.Context) {
	module := c.Param("module")
	collectionName := module + "_data"

	coll := h.storage.GetDynamicCollection(collectionName)
	indexes, err := coll.Indexes().List(context.Background())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer indexes.Close(context.Background())

	var result []bson.M
	if err := indexes.All(context.Background(), &result); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *Handler) DeleteCollectionIndex(c *gin.Context) {
	module := c.Param("module")
	indexName := c.Param("name")

	collectionName := module + "_data"
	coll := h.storage.GetDynamicCollection(collectionName)

	if _, err := coll.Indexes().DropOne(context.Background(), indexName); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Index deleted successfully"})
}
