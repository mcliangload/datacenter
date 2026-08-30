package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"datacenter/internal/middleware"
	"datacenter/internal/model"
	"datacenter/internal/response"
	"datacenter/internal/service"
)

// ItemHandler 数据项与刮削任务接口
type ItemHandler struct {
	items     *service.ItemService
	scrape    *service.ScrapeService
	relations *service.RelationService
}

// NewItemHandler 构造数据项处理器
func NewItemHandler(items *service.ItemService, scrape *service.ScrapeService, relations *service.RelationService) *ItemHandler {
	return &ItemHandler{items: items, scrape: scrape, relations: relations}
}

// Create POST /api/v1/collections/:id/items（操作工）
// body: {path, tags?, auto_scrape?}；auto_scrape 默认 true（刮削添加），false 为直接添加。
func (h *ItemHandler) Create(c *gin.Context) {
	collectionID, ok := parseObjectID(c, "id")
	if !ok {
		return
	}
	var req service.CreateItemReq
	if !bindJSON(c, &req) {
		return
	}
	item, task, err := h.items.Create(c.Request.Context(), middleware.CurrentUserID(c), collectionID, req)
	if err != nil {
		response.Error(c, err)
		return
	}
	data := gin.H{"item": item}
	if task != nil {
		data["task"] = task
	}
	response.OK(c, data)
}

// BatchCreate POST /api/v1/collections/:id/items/batch（批量添加，≤500 条；系统优化 1.1）
func (h *ItemHandler) BatchCreate(c *gin.Context) {
	collectionID, ok := parseObjectID(c, "id")
	if !ok {
		return
	}
	var req service.BatchCreateReq
	if !bindJSON(c, &req) {
		return
	}
	result, err := h.items.BatchCreate(c.Request.Context(), middleware.CurrentUserID(c), collectionID, req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, result)
}

// List GET /api/v1/collections/:id/items?tag=value&tag.gt=...
func (h *ItemHandler) List(c *gin.Context) {
	collectionID, ok := parseObjectID(c, "id")
	if !ok {
		return
	}
	page, pageSize := parsePagination(c)
	items, total, err := h.items.List(c.Request.Context(), middleware.CurrentUserID(c),
		collectionID, c.Request.URL.Query(), page, pageSize)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, pageResult(items, total, page, pageSize))
}

// Get GET /api/v1/items/:itemId
func (h *ItemHandler) Get(c *gin.Context) {
	itemID, ok := parseObjectID(c, "itemId")
	if !ok {
		return
	}
	item, err := h.items.Get(c.Request.Context(), middleware.CurrentUserID(c), itemID, middleware.CurrentRole(c) == model.RoleAdmin)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, item)
}

// Update PATCH /api/v1/items/:itemId（操作工：改标签值/改路径）
func (h *ItemHandler) Update(c *gin.Context) {
	itemID, ok := parseObjectID(c, "itemId")
	if !ok {
		return
	}
	var req service.UpdateItemReq
	if !bindJSON(c, &req) {
		return
	}
	item, err := h.items.Update(c.Request.Context(), middleware.CurrentUserID(c), itemID, req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, item)
}

// Delete DELETE /api/v1/items/:itemId?cascade=&force=&dry_run=（策略化删除，见关联方案 §7）
func (h *ItemHandler) Delete(c *gin.Context) {
	itemID, ok := parseObjectID(c, "itemId")
	if !ok {
		return
	}
	cascade, _ := strconv.ParseBool(c.Query("cascade"))
	force, _ := strconv.ParseBool(c.Query("force"))
	dryRun, _ := strconv.ParseBool(c.Query("dry_run"))
	impact, err := h.relations.DeleteItem(c.Request.Context(), middleware.CurrentUserID(c), itemID, cascade, force, dryRun)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, impact)
}

// Search GET /api/v1/items/search?keyword=&page=&page_size=（可访问集合内按路径搜索，添加关联/选择数据用）
func (h *ItemHandler) Search(c *gin.Context) {
	page, pageSize := parsePagination(c)
	items, total, err := h.items.Search(c.Request.Context(), middleware.CurrentUserID(c),
		middleware.CurrentRole(c) == model.RoleAdmin, c.Query("keyword"), page, pageSize)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, pageResult(items, total, page, pageSize))
}

// Scrape POST /api/v1/items/:itemId/scrape（操作工手动触发刮削）
func (h *ItemHandler) Scrape(c *gin.Context) {
	itemID, ok := parseObjectID(c, "itemId")
	if !ok {
		return
	}
	task, err := h.scrape.Trigger(c.Request.Context(), middleware.CurrentUserID(c), itemID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, task)
}

// ListTasks GET /api/v1/items/:itemId/scrape-tasks
func (h *ItemHandler) ListTasks(c *gin.Context) {
	itemID, ok := parseObjectID(c, "itemId")
	if !ok {
		return
	}
	page, pageSize := parsePagination(c)
	tasks, total, err := h.scrape.ListByItem(c.Request.Context(), middleware.CurrentUserID(c), itemID, page, pageSize)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, pageResult(tasks, total, page, pageSize))
}

// GetTask GET /api/v1/scrape-tasks/:taskId
func (h *ItemHandler) GetTask(c *gin.Context) {
	taskID, ok := parseObjectID(c, "taskId")
	if !ok {
		return
	}
	task, err := h.scrape.GetTask(c.Request.Context(), middleware.CurrentUserID(c), taskID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, task)
}

// GlobalTasks GET /api/v1/scrape-tasks?status=&page=&page_size=（刮削管理页）
func (h *ItemHandler) GlobalTasks(c *gin.Context) {
	page, pageSize := parsePagination(c)
	tasks, total, err := h.scrape.ListTasks(c.Request.Context(),
		middleware.CurrentUserID(c),
		middleware.CurrentRole(c) == model.RoleAdmin,
		c.Query("status"), page, pageSize)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, pageResult(tasks, total, page, pageSize))
}
