package storage

import (
	"context"
	"fmt"
	"time"

	"datacenter/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type mongodbStorage struct {
	client             *mongo.Client
	database           *mongo.Database
	collections        *mongo.Collection
	fieldDefs          *mongo.Collection
	scrapeTasks        *mongo.Collection
	deletedData        *mongo.Collection
	deletedScrapeTasks *mongo.Collection
	dynamicColls       map[string]*mongo.Collection
}

func NewMongoDBStorage(uri, databaseName string) (Storage, error) {
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}

	if err := client.Ping(context.Background(), nil); err != nil {
		return nil, err
	}

	db := client.Database(databaseName)

	storage := &mongodbStorage{
		client:             client,
		database:           db,
		collections:        db.Collection("collections"),
		fieldDefs:          db.Collection("field_definitions"),
		scrapeTasks:        db.Collection("scrape_tasks"),
		deletedData:        db.Collection("deleted_data"),
		deletedScrapeTasks: db.Collection("deleted_scrape_tasks"),
		dynamicColls:       make(map[string]*mongo.Collection),
	}

	return storage, nil
}

func (s *mongodbStorage) GetDynamicCollection(collectionName string) *mongo.Collection {
	if coll, ok := s.dynamicColls[collectionName]; ok {
		return coll
	}
	coll := s.database.Collection(collectionName)
	s.dynamicColls[collectionName] = coll
	return coll
}

func (s *mongodbStorage) CreateDynamicCollection(collectionName string) error {
	err := s.database.CreateCollection(context.Background(), collectionName)
	if err != nil {
		return err
	}
	s.dynamicColls[collectionName] = s.database.Collection(collectionName)
	return nil
}

func (s *mongodbStorage) CreateIndex(collectionName string, keys bson.M, opts *options.IndexOptions) error {
	coll := s.GetDynamicCollection(collectionName)
	_, err := coll.Indexes().CreateOne(context.Background(), mongo.IndexModel{
		Keys:    keys,
		Options: opts,
	})
	return err
}

func (s *mongodbStorage) GetUsersCount() (int64, error) {
	return 0, nil
}

func (s *mongodbStorage) CreateUser(user *models.User) error {
	user.ID = primitive.NewObjectID()
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	_, err := s.database.Collection("users").InsertOne(context.Background(), user)
	return err
}

func (s *mongodbStorage) GetUserByID(id string) (*models.User, error) {
	var user models.User
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}
	err = s.database.Collection("users").FindOne(context.Background(), bson.M{"_id": objID}).Decode(&user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *mongodbStorage) GetUserByUsername(username string) (*models.User, error) {
	var user models.User
	err := s.database.Collection("users").FindOne(context.Background(), bson.M{"username": username}).Decode(&user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *mongodbStorage) UpdateUser(user *models.User) error {
	user.UpdatedAt = time.Now()
	_, err := s.database.Collection("users").UpdateOne(
		context.Background(),
		bson.M{"_id": user.ID},
		bson.M{"$set": user},
	)
	return err
}

func (s *mongodbStorage) DeleteUser(id string) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	_, err = s.database.Collection("users").DeleteOne(context.Background(), bson.M{"_id": objID})
	return err
}

func (s *mongodbStorage) GetUsers(skip, limit int64) ([]models.User, error) {
	opts := options.Find().SetSkip(skip).SetLimit(limit)
	cursor, err := s.database.Collection("users").Find(context.Background(), bson.M{}, opts)
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

func (s *mongodbStorage) GetPermissionsCount() (int64, error) {
	return s.database.Collection("permissions").CountDocuments(context.Background(), bson.M{})
}

func (s *mongodbStorage) CreatePermission(permission *models.Permission) error {
	permission.ID = primitive.NewObjectID()
	permission.CreatedAt = time.Now()
	permission.UpdatedAt = time.Now()
	_, err := s.database.Collection("permissions").InsertOne(context.Background(), permission)
	return err
}

func (s *mongodbStorage) GetPermissionByID(id string) (*models.Permission, error) {
	var permission models.Permission
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}
	err = s.database.Collection("permissions").FindOne(context.Background(), bson.M{"_id": objID}).Decode(&permission)
	if err != nil {
		return nil, err
	}
	return &permission, nil
}

func (s *mongodbStorage) GetPermissionByCode(code string) (*models.Permission, error) {
	var permission models.Permission
	err := s.database.Collection("permissions").FindOne(context.Background(), bson.M{"code": code}).Decode(&permission)
	if err != nil {
		return nil, err
	}
	return &permission, nil
}

func (s *mongodbStorage) GetPermissions(skip, limit int64) ([]models.Permission, error) {
	opts := options.Find().SetSkip(skip).SetLimit(limit)
	cursor, err := s.database.Collection("permissions").Find(context.Background(), bson.M{}, opts)
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

func (s *mongodbStorage) UpdatePermission(permission *models.Permission) error {
	permission.UpdatedAt = time.Now()
	_, err := s.database.Collection("permissions").UpdateOne(
		context.Background(),
		bson.M{"_id": permission.ID},
		bson.M{"$set": permission},
	)
	return err
}

func (s *mongodbStorage) DeletePermission(id string) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	_, err = s.database.Collection("permissions").DeleteOne(context.Background(), bson.M{"_id": objID})
	return err
}

func (s *mongodbStorage) GetRolesCount() (int64, error) {
	return s.database.Collection("roles").CountDocuments(context.Background(), bson.M{})
}

func (s *mongodbStorage) CreateRole(role *models.Role) error {
	role.ID = primitive.NewObjectID()
	role.CreatedAt = time.Now()
	role.UpdatedAt = time.Now()
	_, err := s.database.Collection("roles").InsertOne(context.Background(), role)
	return err
}

func (s *mongodbStorage) GetRoleByID(id string) (*models.Role, error) {
	var role models.Role
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}
	err = s.database.Collection("roles").FindOne(context.Background(), bson.M{"_id": objID}).Decode(&role)
	if err != nil {
		return nil, err
	}
	return &role, nil
}

func (s *mongodbStorage) GetRoleByCode(code string) (*models.Role, error) {
	var role models.Role
	err := s.database.Collection("roles").FindOne(context.Background(), bson.M{"code": code}).Decode(&role)
	if err != nil {
		return nil, err
	}
	return &role, nil
}

func (s *mongodbStorage) GetRoles(skip, limit int64) ([]models.Role, error) {
	opts := options.Find().SetSkip(skip).SetLimit(limit)
	cursor, err := s.database.Collection("roles").Find(context.Background(), bson.M{}, opts)
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

func (s *mongodbStorage) UpdateRole(role *models.Role) error {
	role.UpdatedAt = time.Now()
	_, err := s.database.Collection("roles").UpdateOne(
		context.Background(),
		bson.M{"_id": role.ID},
		bson.M{"$set": role},
	)
	return err
}

func (s *mongodbStorage) DeleteRole(id string) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	_, err = s.database.Collection("roles").DeleteOne(context.Background(), bson.M{"_id": objID})
	return err
}

func (s *mongodbStorage) CreateFieldDefinition(field *models.FieldDefinition) error {
	field.ID = primitive.NewObjectID()
	field.CreatedAt = time.Now()
	field.UpdatedAt = time.Now()
	_, err := s.fieldDefs.InsertOne(context.Background(), field)
	return err
}

func (s *mongodbStorage) GetFieldDefinitionByID(id string) (*models.FieldDefinition, error) {
	var field models.FieldDefinition
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}
	err = s.fieldDefs.FindOne(context.Background(), bson.M{"_id": objID}).Decode(&field)
	if err != nil {
		return nil, err
	}
	return &field, nil
}

func (s *mongodbStorage) GetFieldDefinitionsByModule(module string) ([]models.FieldDefinition, error) {
	cursor, err := s.fieldDefs.Find(context.Background(), bson.M{"module": module})
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

func (s *mongodbStorage) UpdateFieldDefinition(field *models.FieldDefinition) error {
	field.UpdatedAt = time.Now()
	_, err := s.fieldDefs.UpdateOne(
		context.Background(),
		bson.M{"_id": field.ID},
		bson.M{"$set": field},
	)
	return err
}

func (s *mongodbStorage) DeleteFieldDefinition(id string) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	_, err = s.fieldDefs.DeleteOne(context.Background(), bson.M{"_id": objID})
	return err
}

func (s *mongodbStorage) CreateBusinessData(ctx context.Context, data *models.BusinessData) error {
	data.ID = primitive.NewObjectID()
	data.CreatedAt = time.Now()
	data.UpdatedAt = time.Now()

	collectionName := data.Module + "_data"
	coll := s.GetDynamicCollection(collectionName)

	_, err := coll.InsertOne(context.Background(), data)
	return err
}

func (s *mongodbStorage) GetBusinessDataByID(module string, id string) (*models.BusinessData, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	collectionName := module + "_data"
	coll := s.GetDynamicCollection(collectionName)

	var data models.BusinessData
	err = coll.FindOne(context.Background(), bson.M{"_id": objID}).Decode(&data)
	if err != nil {
		return nil, err
	}
	return &data, nil
}

func (s *mongodbStorage) GetBusinessDataByModule(module string, filter bson.M, skip, limit int64) ([]models.BusinessData, error) {
	collectionName := module + "_data"
	coll := s.GetDynamicCollection(collectionName)

	opts := options.Find().SetSkip(skip).SetLimit(limit)
	cursor, err := coll.Find(context.Background(), filter, opts)
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

func (s *mongodbStorage) GetBusinessDataCount(module string, filter bson.M) (int64, error) {
	collectionName := module + "_data"
	coll := s.GetDynamicCollection(collectionName)

	count, err := coll.CountDocuments(context.Background(), filter)
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (s *mongodbStorage) UpdateBusinessData(data *models.BusinessData) error {
	data.UpdatedAt = time.Now()

	collectionName := data.Module + "_data"
	coll := s.GetDynamicCollection(collectionName)

	_, err := coll.UpdateOne(
		context.Background(),
		bson.M{"_id": data.ID},
		bson.M{"$set": data},
	)
	return err
}

func (s *mongodbStorage) DeleteBusinessData(id string, userID string) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	ctx := context.Background()

	// 查找模块信息
	var module string
	// 遍历所有动态集合查找该数据
	for collectionName := range s.dynamicColls {
		coll := s.GetDynamicCollection(collectionName)
		var data models.BusinessData
		err := coll.FindOne(ctx, bson.M{"_id": objID}).Decode(&data)
		if err == nil {
			module = data.Module
			break
		}
	}

	// 如果在缓存的集合中没找到，尝试查找所有集合
	if module == "" {
		// 先找到数据所在的集合和模块
		collections, err := s.collections.Find(ctx, bson.M{})
		if err != nil {
			return err
		}
		defer collections.Close(ctx)

		var colls []models.Collection
		if err := collections.All(ctx, &colls); err != nil {
			return err
		}

		for _, coll := range colls {
			dataColl := s.GetDynamicCollection(coll.CollectionName)
			var data models.BusinessData
			err := dataColl.FindOne(ctx, bson.M{"_id": objID}).Decode(&data)
			if err == nil {
				module = data.Module
				break
			}
		}
	}

	if module == "" {
		return fmt.Errorf("数据不存在或已删除")
	}

	// 再次获取数据用于创建删除记录
	data, err := s.GetBusinessDataByID(module, id)
	if err != nil {
		return err
	}

	// 创建删除记录
	deletedRecord := &models.DeletedData{
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

	_, err = s.deletedData.InsertOne(ctx, deletedRecord)
	if err != nil {
		return err
	}

	// 从原集合中删除数据
	collectionName := module + "_data"
	coll := s.GetDynamicCollection(collectionName)
	_, err = coll.DeleteOne(ctx, bson.M{"_id": objID})
	return err
}

func (s *mongodbStorage) GetDeletedDataByID(id string) (*models.DeletedData, error) {
	var data models.DeletedData
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}
	err = s.deletedData.FindOne(context.Background(), bson.M{"_id": objID}).Decode(&data)
	if err != nil {
		return nil, err
	}
	return &data, nil
}

func (s *mongodbStorage) GetDeletedDataByModule(module string, skip, limit int64) ([]models.DeletedData, error) {
	opts := options.Find().SetSkip(skip).SetLimit(limit).SetSort(bson.M{"deleted_at": -1})
	filter := bson.M{}
	if module != "" {
		filter["module"] = module
	}
	cursor, err := s.deletedData.Find(context.Background(), filter, opts)
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

func (s *mongodbStorage) RecoverDeletedData(id string, userID string) error {
	ctx := context.Background()

	// 获取删除的记录
	deletedRecord, err := s.GetDeletedDataByID(id)
	if err != nil {
		return err
	}

	// 创建恢复的业务数据
	recoveredData := &models.BusinessData{
		BaseModel: models.BaseModel{
			ID:        primitive.NewObjectID(),
			CreatedBy: deletedRecord.CreatedBy,
			CreatedAt: time.Now(),
			UpdatedBy: userID,
			UpdatedAt: time.Now(),
		},
		Module:       deletedRecord.Module,
		Description:  deletedRecord.Description,
		CustomFields: deletedRecord.CustomFields,
		FilePath:     deletedRecord.FilePath,
	}

	collectionName := deletedRecord.Module + "_data"
	coll := s.GetDynamicCollection(collectionName)

	// 确保集合存在
	_, err = coll.InsertOne(ctx, recoveredData)
	if err != nil {
		return err
	}

	// 删除删除记录
	_, err = s.deletedData.DeleteOne(ctx, bson.M{"_id": deletedRecord.ID})
	return err
}

func (s *mongodbStorage) CleanupDeletedData(olderThan time.Time) error {
	_, err := s.deletedData.DeleteMany(context.Background(), bson.M{
		"deleted_at": bson.M{"$lt": olderThan},
	})
	return err
}

func (s *mongodbStorage) CreateScrapeTask(task *models.ScrapeTask) error {
	task.ID = primitive.NewObjectID()
	task.CreatedAt = time.Now()
	task.UpdatedAt = time.Now()
	_, err := s.scrapeTasks.InsertOne(context.Background(), task)
	return err
}

func (s *mongodbStorage) GetScrapeTaskByID(id string) (*models.ScrapeTask, error) {
	var task models.ScrapeTask
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}
	err = s.scrapeTasks.FindOne(context.Background(), bson.M{"_id": objID}).Decode(&task)
	if err != nil {
		return nil, err
	}
	return &task, nil
}

func (s *mongodbStorage) UpdateScrapeTask(task *models.ScrapeTask) error {
	task.UpdatedAt = time.Now()
	_, err := s.scrapeTasks.UpdateOne(
		context.Background(),
		bson.M{"_id": task.ID},
		bson.M{"$set": task},
	)
	return err
}

func (s *mongodbStorage) GetScrapeTasksByModule(module string, status string, skip, limit int64) ([]models.ScrapeTask, error) {
	filter := bson.M{}
	if module != "" {
		filter["module"] = module
	}
	if status != "" {
		filter["status"] = status
	}

	opts := options.Find().SetSkip(skip).SetLimit(limit).SetSort(bson.M{"created_at": -1})
	cursor, err := s.scrapeTasks.Find(context.Background(), filter, opts)
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

func (s *mongodbStorage) GetScrapeTasksCount(module string, filter bson.M) (int64, error) {
	if module != "" {
		filter["module"] = module
	}
	count, err := s.scrapeTasks.CountDocuments(context.Background(), filter)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (s *mongodbStorage) GetDeletedScrapeTasksCount(module string) (int64, error) {
	filter := bson.M{}
	if module != "" {
		filter["module"] = module
	}
	count, err := s.deletedScrapeTasks.CountDocuments(context.Background(), filter)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (s *mongodbStorage) DeleteScrapeTask(id string) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	ctx := context.Background()

	task, err := s.GetScrapeTaskByID(id)
	if err != nil {
		return fmt.Errorf("刮削任务不存在或已删除")
	}

	deletedTask := &models.DeletedScrapeTask{
		BaseModel: models.BaseModel{
			ID:        primitive.NewObjectID(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		Module:         task.Module,
		OriginalID:     task.ID,
		DataPath:       task.DataPath,
		ScraperPath:    task.ScraperPath,
		Status:         task.Status,
		Result:         task.Result,
		ErrorMessage:   task.ErrorMessage,
		StartedAt:      task.StartedAt,
		CompletedAt:    task.CompletedAt,
		BusinessDataID: task.BusinessDataID,
		DeletedAt:      time.Now(),
	}

	_, err = s.deletedScrapeTasks.InsertOne(ctx, deletedTask)
	if err != nil {
		return err
	}

	_, err = s.scrapeTasks.DeleteOne(ctx, bson.M{"_id": objID})
	return err
}

func (s *mongodbStorage) GetDeletedScrapeTasks(module string, skip, limit int64) ([]models.DeletedScrapeTask, error) {
	opts := options.Find().SetSkip(skip).SetLimit(limit).SetSort(bson.M{"deleted_at": -1})
	filter := bson.M{}
	if module != "" {
		filter["module"] = module
	}

	cursor, err := s.deletedScrapeTasks.Find(context.Background(), filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.Background())

	var tasks []models.DeletedScrapeTask
	if err := cursor.All(context.Background(), &tasks); err != nil {
		return nil, err
	}
	return tasks, nil
}

func (s *mongodbStorage) GetDeletedScrapeTaskByID(id string) (*models.DeletedScrapeTask, error) {
	var task models.DeletedScrapeTask
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}
	err = s.deletedScrapeTasks.FindOne(context.Background(), bson.M{"_id": objID}).Decode(&task)
	if err != nil {
		return nil, err
	}
	return &task, nil
}

func (s *mongodbStorage) RecoverScrapeTask(id string) error {
	ctx := context.Background()

	deletedTask, err := s.GetDeletedScrapeTaskByID(id)
	if err != nil {
		return fmt.Errorf("刮削任务不存在或已删除")
	}

	recoveredTask := &models.ScrapeTask{
		BaseModel: models.BaseModel{
			ID:        deletedTask.OriginalID,
			CreatedAt: deletedTask.CreatedAt,
			UpdatedAt: time.Now(),
		},
		Module:         deletedTask.Module,
		DataPath:       deletedTask.DataPath,
		ScraperPath:    deletedTask.ScraperPath,
		Status:         deletedTask.Status,
		Result:         deletedTask.Result,
		ErrorMessage:   deletedTask.ErrorMessage,
		StartedAt:      deletedTask.StartedAt,
		CompletedAt:    deletedTask.CompletedAt,
		BusinessDataID: deletedTask.BusinessDataID,
	}

	_, err = s.scrapeTasks.InsertOne(ctx, recoveredTask)
	if err != nil {
		return err
	}

	objID, _ := primitive.ObjectIDFromHex(id)
	_, err = s.deletedScrapeTasks.DeleteOne(ctx, bson.M{"_id": objID})
	return err
}

func (s *mongodbStorage) CreateCollection(collection *models.Collection) error {
	collection.ID = primitive.NewObjectID()
	collection.CreatedAt = time.Now()
	collection.UpdatedAt = time.Now()

	err := s.CreateDynamicCollection(collection.CollectionName)
	if err != nil {
		return err
	}

	_, err = s.collections.InsertOne(context.Background(), collection)
	return err
}

func (s *mongodbStorage) GetCollectionByModule(module string) (*models.Collection, error) {
	var collection models.Collection
	err := s.collections.FindOne(context.Background(), bson.M{"module": module}).Decode(&collection)
	if err != nil {
		return nil, err
	}
	return &collection, nil
}

func (s *mongodbStorage) GetCollections(skip, limit int64) ([]models.Collection, error) {
	opts := options.Find().SetSkip(skip).SetLimit(limit)
	cursor, err := s.collections.Find(context.Background(), bson.M{}, opts)
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

func (s *mongodbStorage) UpdateCollection(collection *models.Collection) error {
	collection.UpdatedAt = time.Now()
	_, err := s.collections.UpdateOne(
		context.Background(),
		bson.M{"_id": collection.ID},
		bson.M{"$set": collection},
	)
	return err
}

func (s *mongodbStorage) DeleteCollection(module string) error {
	_, err := s.collections.DeleteOne(context.Background(), bson.M{"module": module})
	if err != nil {
		return err
	}

	collectionName := module + "_data"
	delete(s.dynamicColls, collectionName)
	return s.database.Collection(collectionName).Drop(context.Background())
}
