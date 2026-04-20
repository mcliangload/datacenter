package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type BaseModel struct {
	CreatedBy string    `json:"created_by" bson:"created_by"`
	CreatedAt time.Time `json:"created_at" bson:"created_at"`
	UpdatedBy string    `json:"updated_by" bson:"updated_by"`
	UpdatedAt time.Time `json:"updated_at" bson:"updated_at"`
}

type FieldType string

const (
	FieldTypeInt    FieldType = "int"
	FieldTypeFloat  FieldType = "float"
	FieldTypeString FieldType = "string"
	FieldTypeList   FieldType = "list"
)

type FieldDefinition struct {
	ID primitive.ObjectID `json:"_id" bson:"_id"`
	BaseModel
	Module      string      `json:"module" bson:"module"`
	FieldName   string      `json:"field_name" bson:"field_name"`
	FieldType   FieldType   `json:"field_type" bson:"field_type"`
	Description string      `json:"description" bson:"description"`
	Constraints Constraints `json:"constraints" bson:"constraints"`
}

type Constraints struct {
	Min        *float64 `json:"min,omitempty" bson:"min,omitempty"`
	Max        *float64 `json:"max,omitempty" bson:"max,omitempty"`
	MinLength  *int     `json:"min_length,omitempty" bson:"min_length,omitempty"`
	MaxLength  *int     `json:"max_length,omitempty" bson:"max_length,omitempty"`
	ListLength *int     `json:"list_length,omitempty" bson:"list_length,omitempty"`
}

type BusinessData struct {
	ID primitive.ObjectID `json:"_id" bson:"_id"`
	BaseModel
	Module       string                 `json:"module" bson:"module"`
	Description  string                 `json:"description" bson:"description"`
	CustomFields map[string]interface{} `json:"custom_fields" bson:"custom_fields"`
	FilePath     string                 `json:"file_path" bson:"file_path"`
}

type DeletedData struct {
	ID primitive.ObjectID `json:"_id" bson:"_id"`
	BaseModel
	Module       string                 `json:"module" bson:"module"`
	OriginalID   primitive.ObjectID     `json:"original_id" bson:"original_id"`
	Description  string                 `json:"description" bson:"description"`
	CustomFields map[string]interface{} `json:"custom_fields" bson:"custom_fields"`
	FilePath     string                 `json:"file_path" bson:"file_path"`
	DeletedAt    time.Time              `json:"deleted_at" bson:"deleted_at"`
}

type User struct {
	ID primitive.ObjectID `json:"_id" bson:"_id"`
	BaseModel
	Username string   `json:"username" bson:"username"`
	Password string   `json:"password,omitempty" bson:"password"`
	Email    string   `json:"email" bson:"email"`
	RoleIDs  []string `json:"role_ids" bson:"role_ids"`
}

type Permission struct {
	ID primitive.ObjectID `json:"_id" bson:"_id"`
	BaseModel
	Name        string `json:"name" bson:"name"`
	Code        string `json:"code" bson:"code"`
	Description string `json:"description" bson:"description"`
}

type Role struct {
	ID primitive.ObjectID `json:"_id" bson:"_id"`
	BaseModel
	Name          string   `json:"name" bson:"name"`
	Code          string   `json:"code" bson:"code"`
	Description   string   `json:"description" bson:"description"`
	PermissionIDs []string `json:"permission_ids" bson:"permission_ids"`
}

type ScrapeTaskStatus string

const (
	ScrapeTaskStatusPending  ScrapeTaskStatus = "pending"
	ScrapeTaskStatusScraping ScrapeTaskStatus = "scraping"
	ScrapeTaskStatusSuccess  ScrapeTaskStatus = "success"
	ScrapeTaskStatusFailed   ScrapeTaskStatus = "failed"
)

type ScrapeTask struct {
	ID primitive.ObjectID `json:"_id" bson:"_id"`
	BaseModel
	Module         string             `json:"module" bson:"module"`
	DataPath       string             `json:"data_path" bson:"data_path"`
	ScraperPath    string             `json:"scraper_path" bson:"scraper_path"`
	Status         ScrapeTaskStatus   `json:"status" bson:"status"`
	Result         interface{}        `json:"result" bson:"result"`
	ErrorMessage   string             `json:"error_message" bson:"error_message"`
	StartedAt      *time.Time         `json:"started_at" bson:"started_at"`
	CompletedAt    *time.Time         `json:"completed_at" bson:"completed_at"`
	BusinessDataID primitive.ObjectID `json:"business_data_id,omitempty" bson:"business_data_id,omitempty"`
}

type DeletedScrapeTask struct {
	ID primitive.ObjectID `json:"_id" bson:"_id"`
	BaseModel
	Module         string             `json:"module" bson:"module"`
	OriginalID     primitive.ObjectID `json:"original_id" bson:"original_id"`
	DataPath       string             `json:"data_path" bson:"data_path"`
	ScraperPath    string             `json:"scraper_path" bson:"scraper_path"`
	Status         ScrapeTaskStatus   `json:"status" bson:"status"`
	Result         interface{}        `json:"result" bson:"result"`
	ErrorMessage   string             `json:"error_message" bson:"error_message"`
	StartedAt      *time.Time         `json:"started_at" bson:"started_at"`
	CompletedAt    *time.Time         `json:"completed_at" bson:"completed_at"`
	BusinessDataID primitive.ObjectID `json:"business_data_id,omitempty" bson:"business_data_id,omitempty"`
	DeletedAt      time.Time          `json:"deleted_at" bson:"deleted_at"`
}

type Collection struct {
	ID primitive.ObjectID `json:"_id" bson:"_id"`
	BaseModel
	Module         string `json:"module" bson:"module"`
	Description    string `json:"description" bson:"description"`
	DatatypeOwner  string `json:"datatype_owner" bson:"datatype_owner"`
	CollectionName string `json:"collection_name" bson:"collection_name"`
}
