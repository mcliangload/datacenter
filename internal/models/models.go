package models

import (
	"fmt"
	"regexp"
	"strings"
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
	FieldTypeString  FieldType = "string"
	FieldTypeNumber  FieldType = "number"
	FieldTypeBoolean FieldType = "boolean"
	FieldTypeDate    FieldType = "date"
	FieldTypeArray   FieldType = "array"
	FieldTypeObject  FieldType = "object"
)

type ConstraintType string

const (
	ConstraintTypeString  ConstraintType = "string"
	ConstraintTypeNumber  ConstraintType = "number"
	ConstraintTypeEnum    ConstraintType = "enum"
	ConstraintTypePattern ConstraintType = "pattern"
)

type Constraints struct {
	Type       ConstraintType `json:"type" bson:"type"`
	Min        *float64       `json:"min,omitempty" bson:"min,omitempty"`
	Max        *float64       `json:"max,omitempty" bson:"max,omitempty"`
	MinLength  *int           `json:"min_length,omitempty" bson:"min_length,omitempty"`
	MaxLength  *int           `json:"max_length,omitempty" bson:"max_length,omitempty"`
	Pattern    string         `json:"pattern,omitempty" bson:"pattern,omitempty"`
	EnumValues []string       `json:"enum_values,omitempty" bson:"enum_values,omitempty"`
	ListMinLen *int           `json:"list_min_length,omitempty" bson:"list_min_length,omitempty"`
	ListMaxLen *int           `json:"list_max_length,omitempty" bson:"list_max_length,omitempty"`
}

type FieldDefinition struct {
	ID primitive.ObjectID `json:"_id" bson:"_id"`
	BaseModel
	Module       string      `json:"module" bson:"module"`
	FieldName    string      `json:"field_name" bson:"field_name"`
	FieldType    FieldType   `json:"field_type" bson:"field_type"`
	Description  string      `json:"description" bson:"description"`
	Required     bool        `json:"required" bson:"required"`
	DefaultValue interface{} `json:"default_value,omitempty" bson:"default_value,omitempty"`
	Constraints  Constraints `json:"constraints" bson:"constraints"`
}

type FieldValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type FieldValidationResult struct {
	Valid  bool                   `json:"valid"`
	Errors []FieldValidationError `json:"errors,omitempty"`
}

func (f *FieldDefinition) Validate(value interface{}) *FieldValidationResult {
	result := &FieldValidationResult{Valid: true, Errors: []FieldValidationError{}}

	if f.Required && (value == nil || value == "") {
		result.Valid = false
		result.Errors = append(result.Errors, FieldValidationError{
			Field:   f.FieldName,
			Message: "此字段为必填项",
		})
		return result
	}

	if value == nil || value == "" {
		return result
	}

	switch f.FieldType {
	case FieldTypeNumber:
		numVal, ok := value.(float64)
		if !ok {
			intVal, ok := value.(int)
			if !ok {
				result.Valid = false
				result.Errors = append(result.Errors, FieldValidationError{
					Field:   f.FieldName,
					Message: "值必须是数字类型",
				})
				return result
			}
			numVal = float64(intVal)
		}

		if f.Constraints.Min != nil && numVal < *f.Constraints.Min {
			result.Valid = false
			result.Errors = append(result.Errors, FieldValidationError{
				Field:   f.FieldName,
				Message: f.Sprintf("值必须大于等于 %v", *f.Constraints.Min),
			})
		}
		if f.Constraints.Max != nil && numVal > *f.Constraints.Max {
			result.Valid = false
			result.Errors = append(result.Errors, FieldValidationError{
				Field:   f.FieldName,
				Message: f.Sprintf("值必须小于等于 %v", *f.Constraints.Max),
			})
		}

	case FieldTypeString:
		strVal, ok := value.(string)
		if !ok {
			result.Valid = false
			result.Errors = append(result.Errors, FieldValidationError{
				Field:   f.FieldName,
				Message: "值必须是字符串类型",
			})
			return result
		}

		if f.Constraints.MinLength != nil && len(strVal) < *f.Constraints.MinLength {
			result.Valid = false
			result.Errors = append(result.Errors, FieldValidationError{
				Field:   f.FieldName,
				Message: f.Sprintf("字符串长度必须大于等于 %d", *f.Constraints.MinLength),
			})
		}
		if f.Constraints.MaxLength != nil && len(strVal) > *f.Constraints.MaxLength {
			result.Valid = false
			result.Errors = append(result.Errors, FieldValidationError{
				Field:   f.FieldName,
				Message: f.Sprintf("字符串长度必须小于等于 %d", *f.Constraints.MaxLength),
			})
		}
		if f.Constraints.Pattern != "" {
			matched, _ := regexp.MatchString(f.Constraints.Pattern, strVal)
			if !matched {
				result.Valid = false
				result.Errors = append(result.Errors, FieldValidationError{
					Field:   f.FieldName,
					Message: "值不符合正则表达式要求: " + f.Constraints.Pattern,
				})
			}
		}
		if len(f.Constraints.EnumValues) > 0 {
			found := false
			for _, enumVal := range f.Constraints.EnumValues {
				if strVal == enumVal {
					found = true
					break
				}
			}
			if !found {
				result.Valid = false
				result.Errors = append(result.Errors, FieldValidationError{
					Field:   f.FieldName,
					Message: f.Sprintf("值必须是以下之一: %s", strings.Join(f.Constraints.EnumValues, ", ")),
				})
			}
		}

	case FieldTypeBoolean:
		_, ok := value.(bool)
		if !ok {
			result.Valid = false
			result.Errors = append(result.Errors, FieldValidationError{
				Field:   f.FieldName,
				Message: "值必须是布尔类型",
			})
		}

	case FieldTypeDate:
		if strVal, ok := value.(string); ok {
			_, err := time.Parse(time.RFC3339, strVal)
			if err != nil {
				result.Valid = false
				result.Errors = append(result.Errors, FieldValidationError{
					Field:   f.FieldName,
					Message: "日期格式无效，请使用 RFC3339 格式",
				})
			}
		}

	case FieldTypeArray:
		arrVal, ok := value.([]interface{})
		if !ok {
			result.Valid = false
			result.Errors = append(result.Errors, FieldValidationError{
				Field:   f.FieldName,
				Message: "值必须是数组类型",
			})
			return result
		}
		if f.Constraints.ListMinLen != nil && len(arrVal) < *f.Constraints.ListMinLen {
			result.Valid = false
			result.Errors = append(result.Errors, FieldValidationError{
				Field:   f.FieldName,
				Message: f.Sprintf("数组长度必须大于等于 %d", *f.Constraints.ListMinLen),
			})
		}
		if f.Constraints.ListMaxLen != nil && len(arrVal) > *f.Constraints.ListMaxLen {
			result.Valid = false
			result.Errors = append(result.Errors, FieldValidationError{
				Field:   f.FieldName,
				Message: f.Sprintf("数组长度必须小于等于 %d", *f.Constraints.ListMaxLen),
			})
		}
	}

	return result
}

func (f *FieldDefinition) Sprintf(format string, args ...interface{}) string {
	return fmt.Sprintf(format, args...)
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
	Description    string             `json:"description" bson:"description"`
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
