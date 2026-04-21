package scraper

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"time"

	"datacenter/internal/models"
	"datacenter/internal/storage"

	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Scraper 刮削系统接口
type Scraper interface {
	SubmitTask(task *models.ScrapeTask) error
	Start() error
	Stop() error
}

// scraper 刮削系统实现
type scraper struct {
	storage    storage.Storage
	taskQueue  chan *models.ScrapeTask
	workerPool int
	stopChan   chan struct{}
}

// NewScraper 创建刮削系统实例
func NewScraper(storage storage.Storage, workerPool int) Scraper {
	if workerPool <= 0 {
		workerPool = 4 // 默认4个工作协程
	}

	return &scraper{
		storage:    storage,
		taskQueue:  make(chan *models.ScrapeTask, 1000), // 任务队列大小
		workerPool: workerPool,
		stopChan:   make(chan struct{}),
	}
}

// Start 启动刮削系统
func (s *scraper) Start() error {
	log.Info().Msg("启动刮削系统")

	// 启动工作协程
	for i := 0; i < s.workerPool; i++ {
		go s.worker(i)
	}

	log.Info().Int("workers", s.workerPool).Msg("刮削系统启动完成")
	return nil
}

// Stop 停止刮削系统
func (s *scraper) Stop() error {
	log.Info().Msg("停止刮削系统")

	// 关闭停止通道
	close(s.stopChan)

	// 关闭任务队列
	close(s.taskQueue)

	log.Info().Msg("刮削系统已停止")
	return nil
}

// SubmitTask 提交刮削任务
func (s *scraper) SubmitTask(task *models.ScrapeTask) error {
	// 验证任务
	if task.Module == "" || task.DataPath == "" || task.ScraperPath == "" {
		return fmt.Errorf("无效的刮削任务参数")
	}

	// 检查模块是否存在
	_, err := s.storage.GetCollectionByModule(task.Module)
	if err != nil {
		// 如果模块不存在，返回错误
		return fmt.Errorf("模块不存在: %s，请先创建模块集合", task.Module)
	}

	// 保存任务到数据库
	err = s.storage.CreateScrapeTask(task)
	if err != nil {
		return fmt.Errorf("保存刮削任务失败: %v", err)
	}

	// 将任务加入队列
	s.taskQueue <- task
	log.Info().Str("task_id", task.ID.Hex()).Str("module", task.Module).Msg("刮削任务已提交")

	return nil
}

// worker 工作协程
func (s *scraper) worker(id int) {
	log.Info().Int("worker_id", id).Msg("工作协程启动")

	for {
		select {
		case task, ok := <-s.taskQueue:
			if !ok {
				// 任务队列已关闭
				log.Info().Int("worker_id", id).Msg("任务队列已关闭，工作协程退出")
				return
			}
			s.processTask(task, id)

		case <-s.stopChan:
			log.Info().Int("worker_id", id).Msg("收到停止信号，工作协程退出")
			return
		}
	}
}

// processTask 处理刮削任务
func (s *scraper) processTask(task *models.ScrapeTask, workerID int) {
	log.Info().
		Int("worker_id", workerID).
		Str("task_id", task.ID.Hex()).
		Str("module", task.Module).
		Str("data_path", task.DataPath).
		Str("scraper_path", task.ScraperPath).
		Msg("开始处理刮削任务")

	// 更新任务状态为刮削中
	task.Status = models.ScrapeTaskStatusScraping
	now := time.Now()
	task.StartedAt = &now
	err := s.storage.UpdateScrapeTask(task)
	if err != nil {
		log.Error().Str("task_id", task.ID.Hex()).Err(err).Msg("更新任务状态失败")
		return
	}

	// 执行刮削器
	result, err := s.executeScraper(task.ScraperPath, task.DataPath)
	if err != nil {
		// 处理失败情况
		task.Status = models.ScrapeTaskStatusFailed
		task.ErrorMessage = err.Error()
		completedNow := time.Now()
		task.CompletedAt = &completedNow
		err = s.storage.UpdateScrapeTask(task)
		if err != nil {
			log.Error().Str("task_id", task.ID.Hex()).Err(err).Msg("更新任务失败状态失败")
		}
		log.Error().
			Int("worker_id", workerID).
			Str("task_id", task.ID.Hex()).
			Err(err).
			Msg("刮削任务失败")
		return
	}

	// 处理成功情况
	task.Status = models.ScrapeTaskStatusSuccess
	task.Result = result
	completedNow := time.Now()
	task.CompletedAt = &completedNow

	// 存储刮削结果到业务数据
	dataID, err := s.saveScrapedData(task, result)
	if err != nil {
		log.Error().
			Int("worker_id", workerID).
			Str("task_id", task.ID.Hex()).
			Err(err).
			Msg("存储刮削结果失败")
		// 即使存储失败，也更新任务状态
	} else {
		// 更新任务记录，关联业务数据ID
		task.BusinessDataID = dataID
	}

	err = s.storage.UpdateScrapeTask(task)
	if err != nil {
		log.Error().Str("task_id", task.ID.Hex()).Err(err).Msg("更新任务成功状态失败")
		return
	}

	log.Info().
		Int("worker_id", workerID).
		Str("task_id", task.ID.Hex()).
		Str("business_data_id", dataID.Hex()).
		Msg("刮削任务处理完成")
}

// executeScraper 执行刮削器
func (s *scraper) executeScraper(scraperPath, dataPath string) (map[string]interface{}, error) {
	// 检查刮削器文件是否存在
	if _, err := os.Stat(scraperPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("刮削器文件不存在: %s", scraperPath)
	}

	// 检查数据文件是否存在
	if _, err := os.Stat(dataPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("数据文件不存在: %s", dataPath)
	}

	// 执行刮削器
	cmd := exec.Command("python", scraperPath, dataPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("执行刮削器失败: %v, 输出: %s", err, output)
	}

	// 解析刮削器输出
	var result struct {
		Success bool                   `json:"success"`
		Data    map[string]interface{} `json:"data"`
		Error   string                 `json:"error"`
	}

	err = json.Unmarshal(output, &result)
	if err != nil {
		return nil, fmt.Errorf("解析刮削器输出失败: %v, 输出: %s", err, output)
	}

	if !result.Success {
		return nil, fmt.Errorf("刮削器执行失败: %s", result.Error)
	}

	return result.Data, nil
}

// saveScrapedData 存储刮削结果，返回业务数据ID
func (s *scraper) saveScrapedData(task *models.ScrapeTask, data map[string]interface{}) (primitive.ObjectID, error) {
	enhancedData := make(map[string]interface{})
	for k, v := range data {
		enhancedData[k] = v
	}

	enhancedData["scrape_path"] = task.ScraperPath
	enhancedData["data_path"] = task.DataPath
	enhancedData["module"] = task.Module
	enhancedData["task_id"] = task.ID.Hex()
	enhancedData["scraped_at"] = time.Now()

	fieldDefs, err := s.storage.GetFieldDefinitionsByModule(task.Module)
	if err == nil && len(fieldDefs) > 0 {
		for _, fieldDef := range fieldDefs {
			value := enhancedData[fieldDef.FieldName]
			result := fieldDef.Validate(value)
			if !result.Valid {
				log.Warn().
					Str("task_id", task.ID.Hex()).
					Str("field", fieldDef.FieldName).
					Interface("errors", result.Errors).
					Msg("刮削结果字段验证失败，将使用原始值保存")
			}
		}
	}

	businessData := &models.BusinessData{
		Module:       task.Module,
		Description:  fmt.Sprintf("刮削数据 - %s", task.DataPath),
		CustomFields: enhancedData,
		FilePath:     task.DataPath,
		BaseModel: models.BaseModel{
			CreatedBy: task.CreatedBy,
			UpdatedBy: task.CreatedBy,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	collectionName := task.Module + "_data"
	err = s.storage.CreateBusinessData(context.Background(), collectionName, businessData)
	if err != nil {
		return primitive.NilObjectID, fmt.Errorf("存储业务数据失败: %v", err)
	}

	log.Info().
		Str("task_id", task.ID.Hex()).
		Str("module", task.Module).
		Str("data_id", businessData.ID.Hex()).
		Str("collection", task.Module+"_data").
		Msg("刮削结果已存储到对应模块集合")

	return businessData.ID, nil
}
