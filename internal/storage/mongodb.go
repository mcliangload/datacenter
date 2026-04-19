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

	// 刮削任务相关
	CreateScrapeTask(task *models.ScrapeTask) error
	GetScrapeTaskByID(id string) (*models.ScrapeTask, error)
	UpdateScrapeTask(task *models.ScrapeTask) error
	GetScrapeTasksByModule(module string, status string, skip, limit int64) ([]models.ScrapeTask, error)
	DeleteScrapeTask(id string) error

	// 集合管理相关
	CreateCollection(collection *models.Collection) error
	GetCollectionByModule(module string) (*models.Collection, error)
	GetCollections(skip, limit int64) ([]models.Collection, error)
	UpdateCollection(collection *models.Collection) error
	DeleteCollection(module string) error

	// 动态集合管理
	GetDynamicCollection(collectionName string) *mongo.Collection
	CreateDynamicCollection(collectionName string) error
	CreateIndex(collectionName string, keys bson.M, options *options.IndexOptions) error

	// 初始化默认数据
	InitDefaultData() error
}

// mongodbStorage MongoDB存储实现
type mongodbStorage struct {
	client      *mongo.Client
	database    *mongo.Database
	users       *mongo.Collection
	fields      *mongo.Collection
	business    *mongo.Collection
	deleted     *mongo.Collection
	auditLogs   *mongo.Collection
	permissions *mongo.Collection
	roles       *mongo.Collection
	scrapeTasks *mongo.Collection
	collections *mongo.Collection
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
		client:      client,
		database:    db,
		users:       db.Collection("users"),
		fields:      db.Collection("field_definitions"),
		business:    db.Collection("business_data"),
		deleted:     db.Collection("deleted_data"),
		auditLogs:   db.Collection("audit_logs"),
		permissions: db.Collection("permissions"),
		roles:       db.Collection("roles"),
		scrapeTasks: db.Collection("scrape_tasks"),
		collections: db.Collection("collections"),
	}, nil
}

// InitDefaultData 初始化存储
func (s *mongodbStorage) InitDefaultData() error {
	// 只检查连接状态，不创建默认数据
	ctx := context.Background()

	// 测试权限集合是否可访问
	_, err := s.permissions.CountDocuments(ctx, bson.M{})
	if err != nil {
		return err
	}

	// 测试角色集合是否可访问
	_, err = s.roles.CountDocuments(ctx, bson.M{})
	if err != nil {
		return err
	}

	// 测试刮削任务集合是否可访问
	_, err = s.scrapeTasks.CountDocuments(ctx, bson.M{})
	if err != nil {
		return err
	}

	// 测试集合元数据集合是否可访问
	_, err = s.collections.CountDocuments(ctx, bson.M{})
	if err != nil {
		return err
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

	// 使用动态集合
	collectionName := data.Module + "_data"
	collection := s.GetDynamicCollection(collectionName)
	_, err := collection.InsertOne(context.Background(), data)
	return err
}

// GetBusinessDataByID 根据ID获取业务数据
func (s *mongodbStorage) GetBusinessDataByID(id string) (*models.BusinessData, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	// 这里需要先查询所有可能的集合，或者通过其他方式获取模块信息
	// 简化实现：先查询主业务集合
	var data models.BusinessData
	err = s.business.FindOne(context.Background(), bson.M{"_id": objectID}).Decode(&data)
	if err == nil {
		return &data, nil
	}

	// 如果主集合没有，可能在动态集合中
	// 这里需要遍历所有动态集合，实际实现中应该有更好的方法
	return nil, err
}

// GetBusinessDataByModule 根据模块获取业务数据
func (s *mongodbStorage) GetBusinessDataByModule(module string, filter bson.M, skip, limit int64) ([]models.BusinessData, error) {
	filter["module"] = module
	options := options.Find().SetSkip(skip).SetLimit(limit)

	// 使用动态集合
	collectionName := module + "_data"
	collection := s.GetDynamicCollection(collectionName)

	cursor, err := collection.Find(context.Background(), filter, options)
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

	// 使用动态集合
	collectionName := data.Module + "_data"
	collection := s.GetDynamicCollection(collectionName)

	_, err := collection.ReplaceOne(context.Background(), bson.M{"_id": data.ID}, data)
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
		// 尝试从动态集合中获取
		// 简化实现：假设已经知道模块
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
	collectionName := data.Module + "_data"
	collection := s.GetDynamicCollection(collectionName)
	_, err = collection.DeleteOne(context.Background(), bson.M{"_id": objectID})
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
	collectionName := deletedData.Module + "_data"
	collection := s.GetDynamicCollection(collectionName)
	_, err = collection.InsertOne(context.Background(), businessData)
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

// CreateScrapeTask 创建刮削任务
func (s *mongodbStorage) CreateScrapeTask(task *models.ScrapeTask) error {
	task.ID = primitive.NewObjectID()
	task.CreatedAt = time.Now()
	task.UpdatedAt = time.Now()
	_, err := s.scrapeTasks.InsertOne(context.Background(), task)
	return err
}

// GetScrapeTaskByID 根据ID获取刮削任务
func (s *mongodbStorage) GetScrapeTaskByID(id string) (*models.ScrapeTask, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	var task models.ScrapeTask
	err = s.scrapeTasks.FindOne(context.Background(), bson.M{"_id": objectID}).Decode(&task)
	if err != nil {
		return nil, err
	}

	return &task, nil
}

// UpdateScrapeTask 更新刮削任务
func (s *mongodbStorage) UpdateScrapeTask(task *models.ScrapeTask) error {
	task.UpdatedAt = time.Now()
	_, err := s.scrapeTasks.ReplaceOne(context.Background(), bson.M{"_id": task.ID}, task)
	return err
}

// GetScrapeTasksByModule 根据模块获取刮削任务
func (s *mongodbStorage) GetScrapeTasksByModule(module string, status string, skip, limit int64) ([]models.ScrapeTask, error) {
	filter := bson.M{"module": module}
	if status != "" {
		filter["status"] = status
	}
	options := options.Find().SetSkip(skip).SetLimit(limit).SetSort(bson.M{"created_at": -1})

	cursor, err := s.scrapeTasks.Find(context.Background(), filter, options)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.Background())

	var tasks []models.ScrapeTask
	if err := cursor.All(context.Background(), &tasks); err != nil {
		return nil, err
	}

	return tasks, nil
}

// DeleteScrapeTask 删除刮削任务
func (s *mongodbStorage) DeleteScrapeTask(id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	_, err = s.scrapeTasks.DeleteOne(context.Background(), bson.M{"_id": objectID})
	return err
}

// CreateCollection 创建集合
func (s *mongodbStorage) CreateCollection(collection *models.Collection) error {
	collection.ID = primitive.NewObjectID()
	collection.CreatedAt = time.Now()
	collection.UpdatedAt = time.Now()
	_, err := s.collections.InsertOne(context.Background(), collection)
	if err != nil {
		return err
	}

	// 创建对应的动态集合
	err = s.CreateDynamicCollection(collection.CollectionName)
	if err != nil {
		return err
	}

	// 创建默认索引
	err = s.CreateIndex(collection.CollectionName, bson.M{"module": 1}, nil)
	if err != nil {
		return err
	}

	err = s.CreateIndex(collection.CollectionName, bson.M{"created_by": 1}, nil)
	if err != nil {
		return err
	}

	err = s.CreateIndex(collection.CollectionName, bson.M{"created_at": -1}, nil)
	if err != nil {
		return err
	}

	return nil
}

// GetCollectionByModule 根据模块获取集合
func (s *mongodbStorage) GetCollectionByModule(module string) (*models.Collection, error) {
	var collection models.Collection
	err := s.collections.FindOne(context.Background(), bson.M{"module": module}).Decode(&collection)
	if err != nil {
		return nil, err
	}

	return &collection, nil
}

// GetCollections 获取集合列表
func (s *mongodbStorage) GetCollections(skip, limit int64) ([]models.Collection, error) {
	options := options.Find().SetSkip(skip).SetLimit(limit)
	cursor, err := s.collections.Find(context.Background(), bson.M{}, options)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.Background())

	var collections []models.Collection
	if err := cursor.All(context.Background(), &collections); err != nil {
		return nil, err
	}

	return collections, nil
}

// UpdateCollection 更新集合
func (s *mongodbStorage) UpdateCollection(collection *models.Collection) error {
	collection.UpdatedAt = time.Now()
	_, err := s.collections.ReplaceOne(context.Background(), bson.M{"module": collection.Module}, collection)
	return err
}

// DeleteCollection 删除集合
func (s *mongodbStorage) DeleteCollection(module string) error {
	// 检查集合是否存在
	_, err := s.GetCollectionByModule(module)
	if err != nil {
		return err
	}

	// 删除集合元数据
	_, err = s.collections.DeleteOne(context.Background(), bson.M{"module": module})
	if err != nil {
		return err
	}

	// 这里可以选择删除对应的动态集合
	// 但通常不建议直接删除集合，而是保留数据

	return nil
}

// GetDynamicCollection 获取动态集合
func (s *mongodbStorage) GetDynamicCollection(collectionName string) *mongo.Collection {
	return s.database.Collection(collectionName)
}

// CreateDynamicCollection 创建动态集合
func (s *mongodbStorage) CreateDynamicCollection(collectionName string) error {
	// 检查集合是否存在
	collections, err := s.database.ListCollectionNames(context.Background(), bson.M{"name": collectionName})
	if err != nil {
		return err
	}

	if len(collections) == 0 {
		// 创建集合
		err = s.database.CreateCollection(context.Background(), collectionName)
		if err != nil {
			return err
		}
	}

	return nil
}

// CreateIndex 创建索引
func (s *mongodbStorage) CreateIndex(collectionName string, keys bson.M, options *options.IndexOptions) error {
	collection := s.GetDynamicCollection(collectionName)
	_, err := collection.Indexes().CreateOne(context.Background(), mongo.IndexModel{
		Keys:    keys,
		Options: options,
	})
	return err
}
