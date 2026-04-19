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

type RBACStorage interface {
	CreateUser(user *models.User) error
	GetUserByID(id string) (*models.User, error)
	GetUserByUsername(username string) (*models.User, error)
	UpdateUser(user *models.User) error
	DeleteUser(id string) error
	GetUsers(skip, limit int64) ([]models.User, error)
	GetUsersCount() (int64, error)

	CreatePermission(permission *models.Permission) error
	GetPermissionByID(id string) (*models.Permission, error)
	GetPermissionByCode(code string) (*models.Permission, error)
	GetPermissions(skip, limit int64) ([]models.Permission, error)
	GetPermissionsCount() (int64, error)
	UpdatePermission(permission *models.Permission) error
	DeletePermission(id string) error

	CreateRole(role *models.Role) error
	GetRoleByID(id string) (*models.Role, error)
	GetRoleByCode(code string) (*models.Role, error)
	GetRoles(skip, limit int64) ([]models.Role, error)
	GetRolesCount() (int64, error)
	UpdateRole(role *models.Role) error
	DeleteRole(id string) error

	AssignRoleToUser(userID, roleID, operatorID string) error
	RemoveRoleFromUser(userID, roleID string) error
	GetUserRoles(userID string) ([]models.Role, error)

	AssignPermissionToRole(roleID, permissionID, operatorID string) error
	RemovePermissionFromRole(roleID, permissionID string) error
	GetRolePermissions(roleID string) ([]models.Permission, error)

	InitDefaultData() error
}

type rbacMongoDBStorage struct {
	client      *mongo.Client
	database    *mongo.Database
	users       *mongo.Collection
	permissions *mongo.Collection
	roles       *mongo.Collection
}

func NewRBACMongoDBStorage(uri, databaseName string) (RBACStorage, error) {
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}

	if err := client.Ping(context.Background(), nil); err != nil {
		return nil, err
	}

	db := client.Database(databaseName)

	return &rbacMongoDBStorage{
		client:      client,
		database:    db,
		users:       db.Collection("users"),
		permissions: db.Collection("permissions"),
		roles:       db.Collection("roles"),
	}, nil
}

func (s *rbacMongoDBStorage) InitDefaultData() error {
	ctx := context.Background()

	if _, err := s.users.CountDocuments(ctx, bson.M{}); err != nil {
		return err
	}

	if _, err := s.permissions.CountDocuments(ctx, bson.M{}); err != nil {
		return err
	}

	if _, err := s.roles.CountDocuments(ctx, bson.M{}); err != nil {
		return err
	}

	return nil
}

func (s *rbacMongoDBStorage) CreateUser(user *models.User) error {
	user.ID = primitive.NewObjectID()
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	if user.RoleIDs == nil {
		user.RoleIDs = []string{}
	}
	_, err := s.users.InsertOne(context.Background(), user)
	return err
}

func (s *rbacMongoDBStorage) GetUserByID(id string) (*models.User, error) {
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

func (s *rbacMongoDBStorage) GetUserByUsername(username string) (*models.User, error) {
	var user models.User
	err := s.users.FindOne(context.Background(), bson.M{"username": username}).Decode(&user)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (s *rbacMongoDBStorage) UpdateUser(user *models.User) error {
	user.UpdatedAt = time.Now()
	_, err := s.users.ReplaceOne(context.Background(), bson.M{"_id": user.ID}, user)
	return err
}

func (s *rbacMongoDBStorage) DeleteUser(id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	_, err = s.users.DeleteOne(context.Background(), bson.M{"_id": objectID})
	return err
}

func (s *rbacMongoDBStorage) GetUsers(skip, limit int64) ([]models.User, error) {
	opts := options.Find().SetSkip(skip).SetLimit(limit)
	cursor, err := s.users.Find(context.Background(), bson.M{}, opts)
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

func (s *rbacMongoDBStorage) GetUsersCount() (int64, error) {
	return s.users.CountDocuments(context.Background(), bson.M{})
}

func (s *rbacMongoDBStorage) CreatePermission(permission *models.Permission) error {
	permission.ID = primitive.NewObjectID()
	permission.CreatedAt = time.Now()
	permission.UpdatedAt = time.Now()
	_, err := s.permissions.InsertOne(context.Background(), permission)
	return err
}

func (s *rbacMongoDBStorage) GetPermissionByID(id string) (*models.Permission, error) {
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

func (s *rbacMongoDBStorage) GetPermissionByCode(code string) (*models.Permission, error) {
	var permission models.Permission
	err := s.permissions.FindOne(context.Background(), bson.M{"code": code}).Decode(&permission)
	if err != nil {
		return nil, err
	}

	return &permission, nil
}

func (s *rbacMongoDBStorage) GetPermissions(skip, limit int64) ([]models.Permission, error) {
	opts := options.Find().SetSkip(skip).SetLimit(limit)
	cursor, err := s.permissions.Find(context.Background(), bson.M{}, opts)
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

func (s *rbacMongoDBStorage) GetPermissionsCount() (int64, error) {
	return s.permissions.CountDocuments(context.Background(), bson.M{})
}

func (s *rbacMongoDBStorage) UpdatePermission(permission *models.Permission) error {
	permission.UpdatedAt = time.Now()
	_, err := s.permissions.ReplaceOne(context.Background(), bson.M{"_id": permission.ID}, permission)
	return err
}

func (s *rbacMongoDBStorage) DeletePermission(id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	_, err = s.permissions.DeleteOne(context.Background(), bson.M{"_id": objectID})
	return err
}

func (s *rbacMongoDBStorage) CreateRole(role *models.Role) error {
	role.ID = primitive.NewObjectID()
	role.CreatedAt = time.Now()
	role.UpdatedAt = time.Now()
	if role.PermissionIDs == nil {
		role.PermissionIDs = []string{}
	}
	_, err := s.roles.InsertOne(context.Background(), role)
	return err
}

func (s *rbacMongoDBStorage) GetRoleByID(id string) (*models.Role, error) {
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

func (s *rbacMongoDBStorage) GetRoleByCode(code string) (*models.Role, error) {
	var role models.Role
	err := s.roles.FindOne(context.Background(), bson.M{"code": code}).Decode(&role)
	if err != nil {
		return nil, err
	}

	return &role, nil
}

func (s *rbacMongoDBStorage) GetRoles(skip, limit int64) ([]models.Role, error) {
	opts := options.Find().SetSkip(skip).SetLimit(limit)
	cursor, err := s.roles.Find(context.Background(), bson.M{}, opts)
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

func (s *rbacMongoDBStorage) GetRolesCount() (int64, error) {
	return s.roles.CountDocuments(context.Background(), bson.M{})
}

func (s *rbacMongoDBStorage) UpdateRole(role *models.Role) error {
	role.UpdatedAt = time.Now()
	_, err := s.roles.ReplaceOne(context.Background(), bson.M{"_id": role.ID}, role)
	return err
}

func (s *rbacMongoDBStorage) DeleteRole(id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	_, err = s.roles.DeleteOne(context.Background(), bson.M{"_id": objectID})
	return err
}

func (s *rbacMongoDBStorage) AssignRoleToUser(userID, roleID, operatorID string) error {
	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return err
	}

	_, err = s.users.UpdateOne(
		context.Background(),
		bson.M{"_id": objectID},
		bson.M{
			"$addToSet": bson.M{"role_ids": roleID},
			"$set":      bson.M{"updated_by": operatorID, "updated_at": time.Now()},
		},
	)
	return err
}

func (s *rbacMongoDBStorage) RemoveRoleFromUser(userID, roleID string) error {
	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return err
	}

	_, err = s.users.UpdateOne(
		context.Background(),
		bson.M{"_id": objectID},
		bson.M{"$pull": bson.M{"role_ids": roleID}},
	)
	return err
}

func (s *rbacMongoDBStorage) GetUserRoles(userID string) ([]models.Role, error) {
	user, err := s.GetUserByID(userID)
	if err != nil {
		return nil, err
	}

	var roles []models.Role
	for _, roleID := range user.RoleIDs {
		role, err := s.GetRoleByID(roleID)
		if err != nil {
			continue
		}
		roles = append(roles, *role)
	}

	return roles, nil
}

func (s *rbacMongoDBStorage) AssignPermissionToRole(roleID, permissionID, operatorID string) error {
	objectID, err := primitive.ObjectIDFromHex(roleID)
	if err != nil {
		return err
	}

	_, err = s.roles.UpdateOne(
		context.Background(),
		bson.M{"_id": objectID},
		bson.M{
			"$addToSet": bson.M{"permission_ids": permissionID},
			"$set":      bson.M{"updated_by": operatorID, "updated_at": time.Now()},
		},
	)
	return err
}

func (s *rbacMongoDBStorage) RemovePermissionFromRole(roleID, permissionID string) error {
	objectID, err := primitive.ObjectIDFromHex(roleID)
	if err != nil {
		return err
	}

	_, err = s.roles.UpdateOne(
		context.Background(),
		bson.M{"_id": objectID},
		bson.M{"$pull": bson.M{"permission_ids": permissionID}},
	)
	return err
}

func (s *rbacMongoDBStorage) GetRolePermissions(roleID string) ([]models.Permission, error) {
	role, err := s.GetRoleByID(roleID)
	if err != nil {
		return nil, err
	}

	var permissions []models.Permission
	for _, permID := range role.PermissionIDs {
		perm, err := s.GetPermissionByID(permID)
		if err != nil {
			continue
		}
		permissions = append(permissions, *perm)
	}

	return permissions, nil
}
