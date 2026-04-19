package storage

import (
	"context"
	"time"

	"datacenter/internal/models"

	"go.mongodb.org/mongo-driver/bson"
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
	GetUsersCount() (int64, error)

	// 字段定义相关
	CreateFieldDefinition(field *models.FieldDefinition) error
	GetFieldDefinitionByID(id string) (*models.FieldDefinition, error)
	GetFieldDefinitionsByModule(module string) ([]models.FieldDefinition, error)
	UpdateFieldDefinition(field *models.FieldDefinition) error
	DeleteFieldDefinition(id string) error

	// 业务数据相关
	CreateBusinessData(ctx context.Context, data *models.BusinessData) error
	GetBusinessDataByID(module, id string) (*models.BusinessData, error)
	GetBusinessDataByModule(module string, filter bson.M, skip, limit int64) ([]models.BusinessData, error)
	GetBusinessDataCount(module string, filter bson.M) (int64, error)
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
	GetPermissionsCount() (int64, error)
	UpdatePermission(permission *models.Permission) error
	DeletePermission(id string) error

	// 角色相关
	CreateRole(role *models.Role) error
	GetRoleByID(id string) (*models.Role, error)
	GetRoleByCode(code string) (*models.Role, error)
	GetRoles(skip, limit int64) ([]models.Role, error)
	GetRolesCount() (int64, error)
	UpdateRole(role *models.Role) error
	DeleteRole(id string) error

	// 刮削任务相关
	CreateScrapeTask(task *models.ScrapeTask) error
	GetScrapeTaskByID(id string) (*models.ScrapeTask, error)
	UpdateScrapeTask(task *models.ScrapeTask) error
	GetScrapeTasksByModule(module string, status string, skip, limit int64) ([]models.ScrapeTask, error)
	GetScrapeTasksCount(module string, filter bson.M) (int64, error)
	DeleteScrapeTask(id string) error

	// 已删除刮削任务相关
	GetDeletedScrapeTasks(module string, skip, limit int64) ([]models.DeletedScrapeTask, error)
	GetDeletedScrapeTaskByID(id string) (*models.DeletedScrapeTask, error)
	GetDeletedScrapeTasksCount(module string) (int64, error)
	RecoverScrapeTask(id string) error

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
}
