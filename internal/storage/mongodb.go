package storage

import (
	"context"
	"time"

	"datacenter/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Storage 存储接口
type Storage interface {
	// 用户相关
	CreateUser(user *models.User) error
	GetUserByID(id string) (*models.User, error)
	GetUserByUsername(username string) (*models.User, error)
	UpdateUser(user *models.User) error
	DeleteUser(id string) error
	GetUsers(skip, limit int64) ([]models.User, error)

	// 字段定义相关
	CreateFieldDefinition(field *models.FieldDefinition) error
	GetFieldDefinitionByID(id string) (*models.FieldDefinition, error)
	GetFieldDefinitionsByModule(module string) ([]models.FieldDefinition, error)
	UpdateFieldDefinition(field *models.FieldDefinition) error
	DeleteFieldDefinition(id string) error

	// 业务数据相关
	CreateBusinessData(data *models.BusinessData) error
	GetBusinessDataByID(id string) (*models.BusinessData, error)
	GetBusinessDataByModule(module string, filter bson.M, skip, limit int64) ([]models.BusinessData, error)
	UpdateBusinessData(data *models.BusinessData) error
	DeleteBusinessData(id string, userID string) error

	// 已删除数据相关
	GetDeletedDataByID(id string) (*models.DeletedData, error)
	GetDeletedDataByModule(module string, skip, limit int64) ([]models.DeletedData, error)
	RecoverDeletedData(id string, userID string) error
	CleanupDeletedData(olderThan time.Time) error

	// 权限相关
	CreatePermission(permission *models.Permission) error
	GetPermissionByID(id string) (*models.Permission, error)
	GetPermissionByCode(code string) (*models.Permission, error)
	GetPermissions(skip, limit int64) ([]models.Permission, error)
	UpdatePermission(permission *models.Permission) error
	DeletePermission(id string) error

	// 角色相关
	CreateRole(role *models.Role) error
	GetRoleByID(id string) (*models.Role, error)
	GetRoleByCode(code string) (*models.Role, error)
	GetRoles(skip, limit int64) ([]models.Role, error)
	UpdateRole(role *models.Role) error
	DeleteRole(id string) error

	// 用户角色关联
	AssignRoleToUser(userID, roleID, operatorID string) error
	RemoveRoleFromUser(userID, roleID string) error
	GetUserRoles(userID string) ([]models.Role, error)

	// 角色权限关联
	AssignPermissionToRole(roleID, permissionID, operatorID string) error
	RemovePermissionFromRole(roleID, permissionID string) error
	GetRolePermissions(roleID string) ([]models.Permission, error)

	// 初始化默认数据
	InitDefaultData() error
}

// mongodbStorage MongoDB存储实现
type mongodbStorage struct {
	client          *mongo.Client
	database        *mongo.Database
	users           *mongo.Collection
	fields          *mongo.Collection
	business        *mongo.Collection
	deleted         *mongo.Collection
	auditLogs       *mongo.Collection
	permissions     *mongo.Collection
	roles           *mongo.Collection
	userRoles       *mongo.Collection
	rolePermissions *mongo.Collection
}

// NewMongoDBStorage 创建MongoDB存储实例
func NewMongoDBStorage(uri, databaseName string) (Storage, error) {
	// 创建新的客户端连接
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}

	// 测试连接
	if err := client.Ping(context.Background(), nil); err != nil {
		return nil, err
	}

	// 获取指定数据库
	db := client.Database(databaseName)

	// 初始化集合
	return &mongodbStorage{
		client:          client,
		database:        db,
		users:           db.Collection("users"),
		fields:          db.Collection("field_definitions"),
		business:        db.Collection("business_data"),
		deleted:         db.Collection("deleted_data"),
		auditLogs:       db.Collection("audit_logs"),
		permissions:     db.Collection("permissions"),
		roles:           db.Collection("roles"),
		userRoles:       db.Collection("user_roles"),
		rolePermissions: db.Collection("role_permissions"),
	}, nil
}

// InitDefaultData 初始化默认权限和角色
func (s *mongodbStorage) InitDefaultData() error {
	ctx := context.Background()

	// 检查是否已初始化
	count, err := s.permissions.CountDocuments(ctx, bson.M{})
	if err != nil {
		return err
	}
	if count > 0 {
		return nil // 已初始化
	}

	// 创建默认权限
	defaultPermissions := []models.Permission{
		{Name: "管理用户", Code: "manage_users", Description: "管理系统用户"},
		{Name: "管理数据库", Code: "manage_databases", Description: "管理数据库"},
		{Name: "定义字段", Code: "define_fields", Description: "定义自定义字段"},
		{Name: "授予数据所有者", Code: "grant_dataowner", Description: "授予数据所有者权限"},
		{Name: "数据增删改查", Code: "crud_data", Description: "对数据进行增删改查操作"},
	}

	now := time.Now()
	for i := range defaultPermissions {
		defaultPermissions[i].ID = primitive.NewObjectID()
		defaultPermissions[i].CreatedAt = now
		defaultPermissions[i].UpdatedAt = now
		_, err := s.permissions.InsertOne(ctx, defaultPermissions[i])
		if err != nil {
			return err
		}
	}

	// 创建默认角色
	rootPermissions := []string{"manage_users", "manage_databases", "define_fields", "grant_dataowner", "crud_data"}
	ownerPermissions := []string{"define_fields", "grant_dataowner", "crud_data"}
	dataOwnerPermissions := []string{"crud_data"}

	defaultRoles := []models.Role{
		{Name: "超级管理员", Code: "root", Description: "系统最高权限", Permissions: rootPermissions},
		{Name: "数据类型所有者", Code: "datatypeowner", Description: "数据库类型所有者", Permissions: ownerPermissions},
		{Name: "数据所有者", Code: "dataowner", Description: "数据所有者", Permissions: dataOwnerPermissions},
	}

	for i := range defaultRoles {
		defaultRoles[i].ID = primitive.NewObjectID()
		defaultRoles[i].CreatedAt = now
		defaultRoles[i].UpdatedAt = now
		_, err := s.roles.InsertOne(ctx, defaultRoles[i])
		if err != nil {
			return err
		}
	}

	return nil
}

// CreateUser 创建用户
func (s *mongodbStorage) CreateUser(user *models.User) error {
	user.ID = primitive.NewObjectID()
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	_, err := s.users.InsertOne(context.Background(), user)
	return err
}

// GetUserByID 根据ID获取用户
func (s *mongodbStorage) GetUserByID(id string) (*models.User, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	var user models.User
	err = s.users.FindOne(context.Background(), bson.M{"_id": objectID}).Decode(&user)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

// GetUserByUsername 根据用户名获取用户
func (s *mongodbStorage) GetUserByUsername(username string) (*models.User, error) {
	var user models.User
	err := s.users.FindOne(context.Background(), bson.M{"username": username}).Decode(&user)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

// UpdateUser 更新用户
func (s *mongodbStorage) UpdateUser(user *models.User) error {
	user.UpdatedAt = time.Now()
	_, err := s.users.ReplaceOne(context.Background(), bson.M{"_id": user.ID}, user)
	return err
}

// DeleteUser 删除用户
func (s *mongodbStorage) DeleteUser(id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	_, err = s.users.DeleteOne(context.Background(), bson.M{"_id": objectID})
	return err
}

// GetUsers 获取用户列表
func (s *mongodbStorage) GetUsers(skip, limit int64) ([]models.User, error) {
	options := options.Find().SetSkip(skip).SetLimit(limit)
	cursor, err := s.users.Find(context.Background(), bson.M{}, options)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.Background())

	var users []models.User
	if err := cursor.All(context.Background(), &users); err != nil {
		return nil, err
	}

	return users, nil
}

// CreateFieldDefinition 创建字段定义
func (s *mongodbStorage) CreateFieldDefinition(field *models.FieldDefinition) error {
	field.ID = primitive.NewObjectID()
	field.CreatedAt = time.Now()
	field.UpdatedAt = time.Now()
	_, err := s.fields.InsertOne(context.Background(), field)
	return err
}

// GetFieldDefinitionByID 根据ID获取字段定义
func (s *mongodbStorage) GetFieldDefinitionByID(id string) (*models.FieldDefinition, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	var field models.FieldDefinition
	err = s.fields.FindOne(context.Background(), bson.M{"_id": objectID}).Decode(&field)
	if err != nil {
		return nil, err
	}

	return &field, nil
}

// GetFieldDefinitionsByModule 根据模块获取字段定义
func (s *mongodbStorage) GetFieldDefinitionsByModule(module string) ([]models.FieldDefinition, error) {
	cursor, err := s.fields.Find(context.Background(), bson.M{"module": module})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.Background())

	var fields []models.FieldDefinition
	if err := cursor.All(context.Background(), &fields); err != nil {
		return nil, err
	}

	return fields, nil
}

// UpdateFieldDefinition 更新字段定义
func (s *mongodbStorage) UpdateFieldDefinition(field *models.FieldDefinition) error {
	field.UpdatedAt = time.Now()
	_, err := s.fields.ReplaceOne(context.Background(), bson.M{"_id": field.ID}, field)
	return err
}

// DeleteFieldDefinition 删除字段定义
func (s *mongodbStorage) DeleteFieldDefinition(id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	_, err = s.fields.DeleteOne(context.Background(), bson.M{"_id": objectID})
	return err
}

// CreateBusinessData 创建业务数据
func (s *mongodbStorage) CreateBusinessData(data *models.BusinessData) error {
	data.ID = primitive.NewObjectID()
	data.CreatedAt = time.Now()
	data.UpdatedAt = time.Now()
	_, err := s.business.InsertOne(context.Background(), data)
	return err
}

// GetBusinessDataByID 根据ID获取业务数据
func (s *mongodbStorage) GetBusinessDataByID(id string) (*models.BusinessData, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	var data models.BusinessData
	err = s.business.FindOne(context.Background(), bson.M{"_id": objectID}).Decode(&data)
	if err != nil {
		return nil, err
	}

	return &data, nil
}

// GetBusinessDataByModule 根据模块获取业务数据
func (s *mongodbStorage) GetBusinessDataByModule(module string, filter bson.M, skip, limit int64) ([]models.BusinessData, error) {
	filter["module"] = module
	options := options.Find().SetSkip(skip).SetLimit(limit)

	cursor, err := s.business.Find(context.Background(), filter, options)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.Background())

	var dataList []models.BusinessData
	if err := cursor.All(context.Background(), &dataList); err != nil {
		return nil, err
	}

	return dataList, nil
}

// UpdateBusinessData 更新业务数据
func (s *mongodbStorage) UpdateBusinessData(data *models.BusinessData) error {
	data.UpdatedAt = time.Now()
	_, err := s.business.ReplaceOne(context.Background(), bson.M{"_id": data.ID}, data)
	return err
}

// DeleteBusinessData 删除业务数据（软删除）
func (s *mongodbStorage) DeleteBusinessData(id string, userID string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	// 获取原始数据
	var data models.BusinessData
	err = s.business.FindOne(context.Background(), bson.M{"_id": objectID}).Decode(&data)
	if err != nil {
		return err
	}

	// 创建删除记录
	deletedData := models.DeletedData{
		BaseModel: models.BaseModel{
			ID:        primitive.NewObjectID(),
			CreatedBy: userID,
			CreatedAt: time.Now(),
			UpdatedBy: userID,
			UpdatedAt: time.Now(),
		},
		Module:       data.Module,
		OriginalID:   data.ID,
		Description:  data.Description,
		CustomFields: data.CustomFields,
		FilePath:     data.FilePath,
		DeletedAt:    time.Now(),
	}

	// 插入删除记录
	_, err = s.deleted.InsertOne(context.Background(), deletedData)
	if err != nil {
		return err
	}

	// 删除原始数据
	_, err = s.business.DeleteOne(context.Background(), bson.M{"_id": objectID})
	return err
}

// GetDeletedDataByID 根据ID获取已删除数据
func (s *mongodbStorage) GetDeletedDataByID(id string) (*models.DeletedData, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	var data models.DeletedData
	err = s.deleted.FindOne(context.Background(), bson.M{"_id": objectID}).Decode(&data)
	if err != nil {
		return nil, err
	}

	return &data, nil
}

// GetDeletedDataByModule 根据模块获取已删除数据
func (s *mongodbStorage) GetDeletedDataByModule(module string, skip, limit int64) ([]models.DeletedData, error) {
	options := options.Find().SetSkip(skip).SetLimit(limit)

	cursor, err := s.deleted.Find(context.Background(), bson.M{"module": module}, options)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.Background())

	var dataList []models.DeletedData
	if err := cursor.All(context.Background(), &dataList); err != nil {
		return nil, err
	}

	return dataList, nil
}

// RecoverDeletedData 恢复已删除数据
func (s *mongodbStorage) RecoverDeletedData(id string, userID string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	// 获取删除记录
	var deletedData models.DeletedData
	err = s.deleted.FindOne(context.Background(), bson.M{"_id": objectID}).Decode(&deletedData)
	if err != nil {
		return err
	}

	// 创建恢复的数据
	businessData := models.BusinessData{
		BaseModel: models.BaseModel{
			ID:        primitive.NewObjectID(),
			CreatedBy: userID,
			CreatedAt: time.Now(),
			UpdatedBy: userID,
			UpdatedAt: time.Now(),
		},
		Module:       deletedData.Module,
		Description:  deletedData.Description,
		CustomFields: deletedData.CustomFields,
		FilePath:     deletedData.FilePath,
	}

	// 插入恢复的数据
	_, err = s.business.InsertOne(context.Background(), businessData)
	if err != nil {
		return err
	}

	// 删除删除记录
	_, err = s.deleted.DeleteOne(context.Background(), bson.M{"_id": objectID})
	return err
}

// CleanupDeletedData 清理过期的已删除数据
func (s *mongodbStorage) CleanupDeletedData(olderThan time.Time) error {
	_, err := s.deleted.DeleteMany(context.Background(), bson.M{"deleted_at": bson.M{"$lt": olderThan}})
	return err
}

// CreatePermission 创建权限
func (s *mongodbStorage) CreatePermission(permission *models.Permission) error {
	permission.ID = primitive.NewObjectID()
	permission.CreatedAt = time.Now()
	permission.UpdatedAt = time.Now()
	_, err := s.permissions.InsertOne(context.Background(), permission)
	return err
}

// GetPermissionByID 根据ID获取权限
func (s *mongodbStorage) GetPermissionByID(id string) (*models.Permission, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	var permission models.Permission
	err = s.permissions.FindOne(context.Background(), bson.M{"_id": objectID}).Decode(&permission)
	if err != nil {
		return nil, err
	}

	return &permission, nil
}

// GetPermissionByCode 根据Code获取权限
func (s *mongodbStorage) GetPermissionByCode(code string) (*models.Permission, error) {
	var permission models.Permission
	err := s.permissions.FindOne(context.Background(), bson.M{"code": code}).Decode(&permission)
	if err != nil {
		return nil, err
	}

	return &permission, nil
}

// GetPermissions 获取权限列表
func (s *mongodbStorage) GetPermissions(skip, limit int64) ([]models.Permission, error) {
	options := options.Find().SetSkip(skip).SetLimit(limit)
	cursor, err := s.permissions.Find(context.Background(), bson.M{}, options)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.Background())

	var permissions []models.Permission
	if err := cursor.All(context.Background(), &permissions); err != nil {
		return nil, err
	}

	return permissions, nil
}

// UpdatePermission 更新权限
func (s *mongodbStorage) UpdatePermission(permission *models.Permission) error {
	permission.UpdatedAt = time.Now()
	_, err := s.permissions.ReplaceOne(context.Background(), bson.M{"_id": permission.ID}, permission)
	return err
}

// DeletePermission 删除权限
func (s *mongodbStorage) DeletePermission(id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	_, err = s.permissions.DeleteOne(context.Background(), bson.M{"_id": objectID})
	return err
}

// CreateRole 创建角色
func (s *mongodbStorage) CreateRole(role *models.Role) error {
	role.ID = primitive.NewObjectID()
	role.CreatedAt = time.Now()
	role.UpdatedAt = time.Now()
	_, err := s.roles.InsertOne(context.Background(), role)
	return err
}

// GetRoleByID 根据ID获取角色
func (s *mongodbStorage) GetRoleByID(id string) (*models.Role, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	var role models.Role
	err = s.roles.FindOne(context.Background(), bson.M{"_id": objectID}).Decode(&role)
	if err != nil {
		return nil, err
	}

	return &role, nil
}

// GetRoleByCode 根据Code获取角色
func (s *mongodbStorage) GetRoleByCode(code string) (*models.Role, error) {
	var role models.Role
	err := s.roles.FindOne(context.Background(), bson.M{"code": code}).Decode(&role)
	if err != nil {
		return nil, err
	}

	return &role, nil
}

// GetRoles 获取角色列表
func (s *mongodbStorage) GetRoles(skip, limit int64) ([]models.Role, error) {
	options := options.Find().SetSkip(skip).SetLimit(limit)
	cursor, err := s.roles.Find(context.Background(), bson.M{}, options)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.Background())

	var roles []models.Role
	if err := cursor.All(context.Background(), &roles); err != nil {
		return nil, err
	}

	return roles, nil
}

// UpdateRole 更新角色
func (s *mongodbStorage) UpdateRole(role *models.Role) error {
	role.UpdatedAt = time.Now()
	_, err := s.roles.ReplaceOne(context.Background(), bson.M{"_id": role.ID}, role)
	return err
}

// DeleteRole 删除角色
func (s *mongodbStorage) DeleteRole(id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	_, err = s.roles.DeleteOne(context.Background(), bson.M{"_id": objectID})
	return err
}

// AssignRoleToUser 分配角色给用户
func (s *mongodbStorage) AssignRoleToUser(userID, roleID, operatorID string) error {
	userRole := models.UserRole{
		BaseModel: models.BaseModel{
			ID:        primitive.NewObjectID(),
			CreatedBy: operatorID,
			CreatedAt: time.Now(),
			UpdatedBy: operatorID,
			UpdatedAt: time.Now(),
		},
		UserID: userID,
		RoleID: roleID,
	}
	_, err := s.userRoles.InsertOne(context.Background(), userRole)
	return err
}

// RemoveRoleFromUser 从用户移除角色
func (s *mongodbStorage) RemoveRoleFromUser(userID, roleID string) error {
	_, err := s.userRoles.DeleteOne(context.Background(), bson.M{"user_id": userID, "role_id": roleID})
	return err
}

// GetUserRoles 获取用户的角色列表
func (s *mongodbStorage) GetUserRoles(userID string) ([]models.Role, error) {
	// 查询用户的所有角色关联
	cursor, err := s.userRoles.Find(context.Background(), bson.M{"user_id": userID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.Background())

	var userRoles []models.UserRole
	if err := cursor.All(context.Background(), &userRoles); err != nil {
		return nil, err
	}

	// 获取角色详情
	var roles []models.Role
	for _, ur := range userRoles {
		role, err := s.GetRoleByID(ur.RoleID)
		if err != nil {
			continue
		}
		roles = append(roles, *role)
	}

	return roles, nil
}

// AssignPermissionToRole 分配权限给角色
func (s *mongodbStorage) AssignPermissionToRole(roleID, permissionID, operatorID string) error {
	rolePermission := models.RolePermission{
		BaseModel: models.BaseModel{
			ID:        primitive.NewObjectID(),
			CreatedBy: operatorID,
			CreatedAt: time.Now(),
			UpdatedBy: operatorID,
			UpdatedAt: time.Now(),
		},
		RoleID:       roleID,
		PermissionID: permissionID,
	}
	_, err := s.rolePermissions.InsertOne(context.Background(), rolePermission)
	return err
}

// RemovePermissionFromRole 从角色移除权限
func (s *mongodbStorage) RemovePermissionFromRole(roleID, permissionID string) error {
	_, err := s.rolePermissions.DeleteOne(context.Background(), bson.M{"role_id": roleID, "permission_id": permissionID})
	return err
}

// GetRolePermissions 获取角色的权限列表
func (s *mongodbStorage) GetRolePermissions(roleID string) ([]models.Permission, error) {
	// 查询角色的所有权限关联
	cursor, err := s.rolePermissions.Find(context.Background(), bson.M{"role_id": roleID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.Background())

	var rolePermissions []models.RolePermission
	if err := cursor.All(context.Background(), &rolePermissions); err != nil {
		return nil, err
	}

	// 获取权限详情
	var permissions []models.Permission
	for _, rp := range rolePermissions {
		permission, err := s.GetPermissionByID(rp.PermissionID)
		if err != nil {
			continue
		}
		permissions = append(permissions, *permission)
	}

	return permissions, nil
}
