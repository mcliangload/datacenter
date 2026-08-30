package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"datacenter/internal/errno"
	"datacenter/internal/response"
)

// bindJSON 绑定 JSON 请求体：解析失败时已写出 400 响应并返回 false；
// 请求体超过 BodyLimit 上限时返回 413（安全增强 P0-4）。
func bindJSON(c *gin.Context, dst interface{}) bool {
	if err := c.ShouldBindJSON(dst); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			response.Error(c, errno.ErrBodyTooLarge)
		} else {
			response.Error(c, errno.ErrParam.WithCause(err))
		}
		return false
	}
	return true
}

// parseObjectID 解析路径参数为 ObjectID，失败时已写出 400 响应并返回 false
func parseObjectID(c *gin.Context, param string) (primitive.ObjectID, bool) {
	id, err := primitive.ObjectIDFromHex(c.Param(param))
	if err != nil {
		response.Error(c, errno.ErrParam.WithCause(err))
		return primitive.NilObjectID, false
	}
	return id, true
}

// parsePagination 解析分页参数，默认 page=1、page_size=20，上限 200
func parsePagination(c *gin.Context) (page, pageSize int) {
	page, _ = strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ = strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 200 {
		pageSize = 200
	}
	return
}

// pageResult 分页响应体
func pageResult(items interface{}, total int64, page, pageSize int) gin.H {
	return gin.H{
		"items":     items,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	}
}

// parseIntQuery 解析整数查询参数，非法时返回错误
func parseIntQuery(c *gin.Context, key string, def int) (int, error) {
	raw := c.Query(key)
	if raw == "" {
		return def, nil
	}
	return strconv.Atoi(raw)
}

// splitComma 按逗号切分查询参数
func splitComma(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if s := strings.TrimSpace(p); s != "" {
			out = append(out, s)
		}
	}
	return out
}
