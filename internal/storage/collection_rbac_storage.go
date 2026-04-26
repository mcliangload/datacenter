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

type CollectionRBACStorage interface {
	CreateCollectionRole(role *models.CollectionRole) error
	GetCollectionRoleByID(id string) (*models.CollectionRole, error)
	GetCollectionRolesByModule(module string) ([]models.CollectionRole, error)
	UpdateCollectionRole(role *models.CollectionRole) error
	DeleteCollectionRole(id string) error
	GetCollectionRoleByType(module, roleType string) (*models.CollectionRole, error)

	AssignCollectionRole(assignment *models.CollectionRoleAssignment) error
	RemoveCollectionRoleAssignment(userID, module, roleID string) error
	GetCollectionRoleAssignments(module string) ([]models.CollectionRoleAssignment, error)
	GetUserCollectionRoles(userID string) ([]models.CollectionRoleAssignment, error)
	GetUserCollectionRole(userID, module string) (*models.CollectionRoleAssignment, error)

	CreateAuditLog(log *models.AuditLog) error
	GetAuditLogs(filter bson.M, skip, limit int64) ([]models.AuditLog, error)
	GetAuditLogsByUser(userID string, skip, limit int64) ([]models.AuditLog, error)
	GetAuditLogsByResource(resource, resourceID string, skip, limit int64) ([]models.AuditLog, error)
}

type collectionRBACStorage struct {
	client                    *mongo.Client
	database                  *mongo.Database
	collectionRoles           *mongo.Collection
	collectionRoleAssignments *mongo.Collection
	auditLogs                 *mongo.Collection
}

func NewCollectionRBACStorage(uri, databaseName string) (CollectionRBACStorage, error) {
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}

	if err := client.Ping(context.Background(), nil); err != nil {
		return nil, err
	}

	db := client.Database(databaseName)

	return &collectionRBACStorage{
		client:                    client,
		database:                  db,
		collectionRoles:           db.Collection("collection_roles"),
		collectionRoleAssignments: db.Collection("collection_role_assignments"),
		auditLogs:                 db.Collection("audit_logs"),
	}, nil
}

func (s *collectionRBACStorage) CreateCollectionRole(role *models.CollectionRole) error {
	role.ID = primitive.NewObjectID()
	role.CreatedAt = time.Now()
	role.UpdatedAt = time.Now()
	if role.PermissionIDs == nil {
		role.PermissionIDs = []string{}
	}
	_, err := s.collectionRoles.InsertOne(context.Background(), role)
	return err
}

func (s *collectionRBACStorage) GetCollectionRoleByID(id string) (*models.CollectionRole, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	var role models.CollectionRole
	err = s.collectionRoles.FindOne(context.Background(), bson.M{"_id": objectID}).Decode(&role)
	if err != nil {
		return nil, err
	}

	return &role, nil
}

func (s *collectionRBACStorage) GetCollectionRolesByModule(module string) ([]models.CollectionRole, error) {
	cursor, err := s.collectionRoles.Find(context.Background(), bson.M{"collection_module": module})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.Background())

	var roles []models.CollectionRole
	if err := cursor.All(context.Background(), &roles); err != nil {
		return nil, err
	}

	return roles, nil
}

func (s *collectionRBACStorage) UpdateCollectionRole(role *models.CollectionRole) error {
	role.UpdatedAt = time.Now()
	_, err := s.collectionRoles.ReplaceOne(context.Background(), bson.M{"_id": role.ID}, role)
	return err
}

func (s *collectionRBACStorage) DeleteCollectionRole(id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	_, err = s.collectionRoles.DeleteOne(context.Background(), bson.M{"_id": objectID})
	return err
}

func (s *collectionRBACStorage) GetCollectionRoleByType(module, roleType string) (*models.CollectionRole, error) {
	var role models.CollectionRole
	err := s.collectionRoles.FindOne(context.Background(), bson.M{"collection_module": module, "type": roleType}).Decode(&role)
	if err != nil {
		return nil, err
	}

	return &role, nil
}

func (s *collectionRBACStorage) AssignCollectionRole(assignment *models.CollectionRoleAssignment) error {
	ctx := context.Background()

	existing, err := s.GetUserCollectionRole(assignment.UserID, assignment.CollectionModule)
	if err != nil && err != mongo.ErrNoDocuments {
		return err
	}

	if existing != nil {
		if existing.CollectionRoleID == assignment.CollectionRoleID {
			return nil
		}
		assignment.ID = existing.ID
		assignment.CreatedAt = existing.CreatedAt
		assignment.CreatedBy = existing.CreatedBy
		assignment.UpdatedAt = time.Now()
		_, err = s.collectionRoleAssignments.ReplaceOne(ctx, bson.M{"_id": existing.ID}, assignment)
		return err
	}

	assignment.ID = primitive.NewObjectID()
	assignment.CreatedAt = time.Now()
	assignment.UpdatedAt = time.Now()
	_, err = s.collectionRoleAssignments.InsertOne(ctx, assignment)
	return err
}

func (s *collectionRBACStorage) RemoveCollectionRoleAssignment(userID, module, roleID string) error {
	_, err := s.collectionRoleAssignments.DeleteOne(
		context.Background(),
		bson.M{"user_id": userID, "collection_module": module, "collection_role_id": roleID},
	)
	return err
}

func (s *collectionRBACStorage) GetCollectionRoleAssignments(module string) ([]models.CollectionRoleAssignment, error) {
	cursor, err := s.collectionRoleAssignments.Find(context.Background(), bson.M{"collection_module": module})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.Background())

	var assignments []models.CollectionRoleAssignment
	if err := cursor.All(context.Background(), &assignments); err != nil {
		return nil, err
	}

	return assignments, nil
}

func (s *collectionRBACStorage) GetUserCollectionRoles(userID string) ([]models.CollectionRoleAssignment, error) {
	cursor, err := s.collectionRoleAssignments.Find(context.Background(), bson.M{"user_id": userID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.Background())

	var assignments []models.CollectionRoleAssignment
	if err := cursor.All(context.Background(), &assignments); err != nil {
		return nil, err
	}

	return assignments, nil
}

func (s *collectionRBACStorage) GetUserCollectionRole(userID, module string) (*models.CollectionRoleAssignment, error) {
	var assignment models.CollectionRoleAssignment
	err := s.collectionRoleAssignments.FindOne(
		context.Background(),
		bson.M{"user_id": userID, "collection_module": module},
	).Decode(&assignment)
	if err != nil {
		return nil, err
	}

	return &assignment, nil
}

func (s *collectionRBACStorage) CreateAuditLog(log *models.AuditLog) error {
	log.ID = primitive.NewObjectID()
	if log.Timestamp.IsZero() {
		log.Timestamp = time.Now()
	}
	_, err := s.auditLogs.InsertOne(context.Background(), log)
	return err
}

func (s *collectionRBACStorage) GetAuditLogs(filter bson.M, skip, limit int64) ([]models.AuditLog, error) {
	opts := options.Find().
		SetSkip(skip).
		SetLimit(limit).
		SetSort(bson.M{"timestamp": -1})
	cursor, err := s.auditLogs.Find(context.Background(), filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.Background())

	var logs []models.AuditLog
	if err := cursor.All(context.Background(), &logs); err != nil {
		return nil, err
	}

	return logs, nil
}

func (s *collectionRBACStorage) GetAuditLogsByUser(userID string, skip, limit int64) ([]models.AuditLog, error) {
	return s.GetAuditLogs(bson.M{"user_id": userID}, skip, limit)
}

func (s *collectionRBACStorage) GetAuditLogsByResource(resource, resourceID string, skip, limit int64) ([]models.AuditLog, error) {
	return s.GetAuditLogs(bson.M{"resource": resource, "resource_id": resourceID}, skip, limit)
}
