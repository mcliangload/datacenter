package api

import (
	"net/http"
	"strconv"

	"datacenter/internal/auth"
	"datacenter/internal/models"
	"datacenter/internal/storage"
	"datacenter/pkg/jql"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Handler API处理器
type Handler struct {
	storage     storage.Storage
	userStorage storage.Storage
}

// NewHandler 创建API处理器实例
func NewHandler(storage, userStorage storage.Storage) *Handler {
	return &Handler{
		storage:     storage,
		userStorage: userStorage,
	}
}

// RegisterRoutes 注册API路由
func (h *Handler) RegisterRoutes(router *gin.Engine, jwtService auth.JWTService) {
	// 公开路由
	public := router.Group("/api")
	{
		// 存储jwtService到上下文中
		public.Use(func(c *gin.Context) {
			c.Set("jwtService", jwtService)
			c.Next()
		})
		public.POST("/auth/login", h.Login)
	}

	// 受保护路由
	protected := router.Group("/api")
	protected.Use(auth.AuthMiddleware(jwtService)) // 添加认证中间件
	{
		// 用户管理
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

		// 权限管理
		permissions := protected.Group("/permissions")
		{
			permissions.POST("", h.CreatePermission)
			permissions.GET("", h.GetPermissions)
			permissions.GET("/:id", h.GetPermissionByID)
			permissions.PUT("/:id", h.UpdatePermission)
			permissions.DELETE("/:id", h.DeletePermission)
		}

		// 角色管理
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

		// 字段定义管理
		fields := protected.Group("/fields")
		{
			fields.POST("", h.CreateFieldDefinition)
			fields.GET("/module/:module", h.GetFieldDefinitionsByModule)
			fields.GET("/:id", h.GetFieldDefinitionByID)
			fields.PUT("/:id", h.UpdateFieldDefinition)
			fields.DELETE("/:id", h.DeleteFieldDefinition)
		}

		// 业务数据管理
		business := protected.Group("/business")
		{
			business.POST("", h.CreateBusinessData)
			business.GET("/module/:module", h.GetBusinessDataByModule)
			business.GET("/:id", h.GetBusinessDataByID)
			business.PUT("/:id", h.UpdateBusinessData)
			business.DELETE("/:id", h.DeleteBusinessData)
		}

		// 已删除数据管理
		deleted := protected.Group("/deleted")
		{
			deleted.GET("/module/:module", h.GetDeletedDataByModule)
			deleted.GET("/:id", h.GetDeletedDataByID)
			deleted.POST("/:id/recover", h.RecoverDeletedData)
		}
	}
}

// Login 用户登录
func (h *Handler) Login(c *gin.Context) {
	var loginReq struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&loginReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 从用户存储中获取用户
	user, err := h.userStorage.GetUserByUsername(loginReq.Username)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
		return
	}

	// 验证密码
	if err := auth.CheckPassword(loginReq.Password, user.Password); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
		return
	}

	// 获取用户的角色和权限
	roles, _ := h.userStorage.GetUserRoles(user.ID.Hex())
	roleCodes := make([]string, len(roles))
	for i, role := range roles {
		roleCodes[i] = role.Code
	}

	// 生成JWT Token
	jwtService := c.MustGet("jwtService").(auth.JWTService)
	token, err := jwtService.GenerateToken(user.ID.Hex(), roleCodes, user.Permissions)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user": gin.H{
			"id":          user.ID.Hex(),
			"username":    user.Username,
			"email":       user.Email,
			"roles":       roleCodes,
			"permissions": user.Permissions,
		},
	})
}

// CreateUser 创建用户
func (h *Handler) CreateUser(c *gin.Context) {
	var user models.User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 加密密码
	hashedPassword, err := auth.HashPassword(user.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}
	user.Password = hashedPassword

	// 获取当前用户ID
	userID := c.GetString("userID")
	user.CreatedBy = userID
	user.UpdatedBy = userID

	if err := h.userStorage.CreateUser(&user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 分配角色
	for _, roleCode := range user.Roles {
		role, err := h.userStorage.GetRoleByCode(roleCode)
		if err != nil {
			continue
		}
		h.userStorage.AssignRoleToUser(user.ID.Hex(), role.ID.Hex(), userID)
	}

	// 清除密码字段
	user.Password = ""

	c.JSON(http.StatusCreated, user)
}

// GetUsers 获取用户列表
func (h *Handler) GetUsers(c *gin.Context) {
	skip, _ := strconv.ParseInt(c.DefaultQuery("skip", "0"), 10, 64)
	limit, _ := strconv.ParseInt(c.DefaultQuery("limit", "10"), 10, 64)

	users, err := h.userStorage.GetUsers(skip, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 清除密码字段
	for i := range users {
		users[i].Password = ""
	}

	c.JSON(http.StatusOK, users)
}

// GetUserByID 根据ID获取用户
func (h *Handler) GetUserByID(c *gin.Context) {
	id := c.Param("id")
	user, err := h.userStorage.GetUserByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// 获取用户的角色
	roles, _ := h.userStorage.GetUserRoles(id)
	roleCodes := make([]string, len(roles))
	for i, role := range roles {
		roleCodes[i] = role.Code
	}
	user.Roles = roleCodes

	// 清除密码字段
	user.Password = ""

	c.JSON(http.StatusOK, user)
}

// UpdateUser 更新用户
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

	// 如果密码被修改，加密新密码
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

	if err := h.userStorage.UpdateUser(&user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 清除密码字段
	user.Password = ""

	c.JSON(http.StatusOK, user)
}

// DeleteUser 删除用户
func (h *Handler) DeleteUser(c *gin.Context) {
	id := c.Param("id")
	if err := h.userStorage.DeleteUser(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User deleted successfully"})
}

// AssignRoleToUser 分配角色给用户
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
	if err := h.userStorage.AssignRoleToUser(userID, req.RoleID, operatorID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Role assigned successfully"})
}

// RemoveRoleFromUser 从用户移除角色
func (h *Handler) RemoveRoleFromUser(c *gin.Context) {
	userID := c.Param("id")
	roleID := c.Param("roleId")

	if err := h.userStorage.RemoveRoleFromUser(userID, roleID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Role removed successfully"})
}

// GetUserRoles 获取用户的角色列表
func (h *Handler) GetUserRoles(c *gin.Context) {
	userID := c.Param("id")

	roles, err := h.userStorage.GetUserRoles(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, roles)
}

// CreatePermission 创建权限
func (h *Handler) CreatePermission(c *gin.Context) {
	var permission models.Permission
	if err := c.ShouldBindJSON(&permission); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetString("userID")
	permission.CreatedBy = userID
	permission.UpdatedBy = userID

	if err := h.userStorage.CreatePermission(&permission); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, permission)
}

// GetPermissions 获取权限列表
func (h *Handler) GetPermissions(c *gin.Context) {
	skip, _ := strconv.ParseInt(c.DefaultQuery("skip", "0"), 10, 64)
	limit, _ := strconv.ParseInt(c.DefaultQuery("limit", "10"), 10, 64)

	permissions, err := h.userStorage.GetPermissions(skip, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, permissions)
}

// GetPermissionByID 根据ID获取权限
func (h *Handler) GetPermissionByID(c *gin.Context) {
	id := c.Param("id")
	permission, err := h.userStorage.GetPermissionByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Permission not found"})
		return
	}

	c.JSON(http.StatusOK, permission)
}

// UpdatePermission 更新权限
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

	if err := h.userStorage.UpdatePermission(&permission); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, permission)
}

// DeletePermission 删除权限
func (h *Handler) DeletePermission(c *gin.Context) {
	id := c.Param("id")
	if err := h.userStorage.DeletePermission(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Permission deleted successfully"})
}

// CreateRole 创建角色
func (h *Handler) CreateRole(c *gin.Context) {
	var role models.Role
	if err := c.ShouldBindJSON(&role); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetString("userID")
	role.CreatedBy = userID
	role.UpdatedBy = userID

	if err := h.userStorage.CreateRole(&role); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 分配权限
	for _, permissionCode := range role.Permissions {
		permission, err := h.userStorage.GetPermissionByCode(permissionCode)
		if err != nil {
			continue
		}
		h.userStorage.AssignPermissionToRole(role.ID.Hex(), permission.ID.Hex(), userID)
	}

	c.JSON(http.StatusCreated, role)
}

// GetRoles 获取角色列表
func (h *Handler) GetRoles(c *gin.Context) {
	skip, _ := strconv.ParseInt(c.DefaultQuery("skip", "0"), 10, 64)
	limit, _ := strconv.ParseInt(c.DefaultQuery("limit", "10"), 10, 64)

	roles, err := h.userStorage.GetRoles(skip, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, roles)
}

// GetRoleByID 根据ID获取角色
func (h *Handler) GetRoleByID(c *gin.Context) {
	id := c.Param("id")
	role, err := h.userStorage.GetRoleByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Role not found"})
		return
	}

	c.JSON(http.StatusOK, role)
}

// UpdateRole 更新角色
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

	if err := h.userStorage.UpdateRole(&role); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, role)
}

// DeleteRole 删除角色
func (h *Handler) DeleteRole(c *gin.Context) {
	id := c.Param("id")
	if err := h.userStorage.DeleteRole(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Role deleted successfully"})
}

// AssignPermissionToRole 分配权限给角色
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
	if err := h.userStorage.AssignPermissionToRole(roleID, req.PermissionID, operatorID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Permission assigned to role successfully"})
}

// RemovePermissionFromRole 从角色移除权限
func (h *Handler) RemovePermissionFromRole(c *gin.Context) {
	roleID := c.Param("id")
	permissionID := c.Param("permissionId")

	if err := h.userStorage.RemovePermissionFromRole(roleID, permissionID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Permission removed from role successfully"})
}

// GetRolePermissions 获取角色的权限列表
func (h *Handler) GetRolePermissions(c *gin.Context) {
	roleID := c.Param("id")

	permissions, err := h.userStorage.GetRolePermissions(roleID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, permissions)
}

// CreateFieldDefinition 创建字段定义
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

// GetFieldDefinitionsByModule 根据模块获取字段定义
func (h *Handler) GetFieldDefinitionsByModule(c *gin.Context) {
	module := c.Param("module")
	fields, err := h.storage.GetFieldDefinitionsByModule(module)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, fields)
}

// GetFieldDefinitionByID 根据ID获取字段定义
func (h *Handler) GetFieldDefinitionByID(c *gin.Context) {
	id := c.Param("id")
	field, err := h.storage.GetFieldDefinitionByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Field definition not found"})
		return
	}

	c.JSON(http.StatusOK, field)
}

// UpdateFieldDefinition 更新字段定义
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

// DeleteFieldDefinition 删除字段定义
func (h *Handler) DeleteFieldDefinition(c *gin.Context) {
	id := c.Param("id")
	if err := h.storage.DeleteFieldDefinition(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Field definition deleted successfully"})
}

// CreateBusinessData 创建业务数据
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

// GetBusinessDataByModule 根据模块获取业务数据
func (h *Handler) GetBusinessDataByModule(c *gin.Context) {
	module := c.Param("module")
	skip, _ := strconv.ParseInt(c.DefaultQuery("skip", "0"), 10, 64)
	limit, _ := strconv.ParseInt(c.DefaultQuery("limit", "10"), 10, 64)

	// 实现JQL查询转换
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

// GetBusinessDataByID 根据ID获取业务数据
func (h *Handler) GetBusinessDataByID(c *gin.Context) {
	id := c.Param("id")
	data, err := h.storage.GetBusinessDataByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Business data not found"})
		return
	}

	c.JSON(http.StatusOK, data)
}

// UpdateBusinessData 更新业务数据
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

// DeleteBusinessData 删除业务数据
func (h *Handler) DeleteBusinessData(c *gin.Context) {
	id := c.Param("id")
	userID := c.GetString("userID")

	if err := h.storage.DeleteBusinessData(id, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Business data deleted successfully"})
}

// GetDeletedDataByModule 根据模块获取已删除数据
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

// GetDeletedDataByID 根据ID获取已删除数据
func (h *Handler) GetDeletedDataByID(c *gin.Context) {
	id := c.Param("id")
	data, err := h.storage.GetDeletedDataByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Deleted data not found"})
		return
	}

	c.JSON(http.StatusOK, data)
}

// RecoverDeletedData 恢复已删除数据
func (h *Handler) RecoverDeletedData(c *gin.Context) {
	id := c.Param("id")
	userID := c.GetString("userID")

	if err := h.storage.RecoverDeletedData(id, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Deleted data recovered successfully"})
}
