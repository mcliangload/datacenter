package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"datacenter/internal/logger"
	"datacenter/internal/models"
	"datacenter/pkg/rbac"

	"github.com/gin-gonic/gin"
)

func CollectionPermissionMiddleware(collectionRBACService *rbac.CollectionRBACService, requiredPermission rbac.CollectionPermission) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{"error": "No user ID found"})
			c.Abort()
			return
		}

		module := c.Param("module")
		if module == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Collection module is required"})
			c.Abort()
			return
		}
		logger.Error("module is :%s, requiredPermission is :%v", module, requiredPermission)

		hasPermission, err := collectionRBACService.CheckCollectionPermission(
			c.Request.Context(),
			userID.(string),
			module,
			requiredPermission,
		)
		logger.Error("hasPermission is :%v, err is :%v", hasPermission, err)
		if err != nil || !hasPermission {
			c.JSON(http.StatusForbidden, gin.H{"error": "Permission denied: " + string(requiredPermission)})
			c.Abort()
			return
		}

		c.Next()
	}
}

// CollectionPermissionMiddlewareFromBody extracts the module from the JSON request body
// and checks the specified collection-level permission. This is needed for POST routes
// where the module is in the body rather than the URL.
func CollectionPermissionMiddlewareFromBody(collectionRBACService *rbac.CollectionRBACService, requiredPermission rbac.CollectionPermission) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{"error": "No user ID found"})
			c.Abort()
			return
		}

		bodyBytes, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot read request body"})
			c.Abort()
			return
		}
		c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		var body struct {
			Module string `json:"module"`
		}
		if err := json.Unmarshal(bodyBytes, &body); err != nil || body.Module == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Module is required in request body"})
			c.Abort()
			return
		}

		hasPermission, err := collectionRBACService.CheckCollectionPermission(
			c.Request.Context(),
			userID.(string),
			body.Module,
			requiredPermission,
		)
		if err != nil || !hasPermission {
			c.JSON(http.StatusForbidden, gin.H{"error": "Permission denied: " + string(requiredPermission)})
			c.Abort()
			return
		}

		c.Next()
	}
}

// CollectionPermissionFieldAdminMiddleware checks that the user has collection:field:admin
// permission for the field's module. It works for PUT/DELETE by fetching the field record first.
func CollectionPermissionFieldAdminMiddleware(collectionRBACService *rbac.CollectionRBACService, storage interface {
	GetFieldDefinitionByID(id string) (*models.FieldDefinition, error)
}) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{"error": "No user ID found"})
			c.Abort()
			return
		}

		fieldID := c.Param("id")
		if fieldID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Field ID is required"})
			c.Abort()
			return
		}

		field, err := storage.GetFieldDefinitionByID(fieldID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Field definition not found"})
			c.Abort()
			return
		}

		hasPermission, err := collectionRBACService.CheckCollectionPermission(
			c.Request.Context(),
			userID.(string),
			field.Module,
			rbac.CollectionPermissionFieldAdmin,
		)
		if err != nil || !hasPermission {
			c.JSON(http.StatusForbidden, gin.H{"error": "Permission denied: " + string(rbac.CollectionPermissionFieldAdmin)})
			c.Abort()
			return
		}

		c.Next()
	}
}
