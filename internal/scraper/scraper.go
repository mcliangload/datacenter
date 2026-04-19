package scraper

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"time"

	"datacenter/internal/models"
	"datacenter/internal/storage"

	"github.com/rs/zerolog/log"
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
		// 如果模块不存在，创建默认集合
		collection := &models.Collection{
			Module:         task.Module,
			Description:    fmt.Sprintf("%s模块数据", task.Module),
			DatatypeOwner:  task.CreatedBy,
			CollectionName: task.Module + "_data",
		}
		err = s.storage.CreateCollection(collection)
		if err != nil {
			return fmt.Errorf("创建模块集合失败: %v", err)
		}
		log.Info().Str("module", task.Module).Msg("自动创建模块集合")
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
	err = s.storage.UpdateScrapeTask(task)
	if err != nil {
		log.Error().Str("task_id", task.ID.Hex()).Err(err).Msg("更新任务成功状态失败")
		return
	}

	// 存储刮削结果到业务数据
	err = s.saveScrapedData(task, result)
	if err != nil {
		log.Error().
			Int("worker_id", workerID).
			Str("task_id", task.ID.Hex()).
			Err(err).
			Msg("存储刮削结果失败")
	}

	log.Info().
		Int("worker_id", workerID).
		Str("task_id", task.ID.Hex()).
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

// saveScrapedData 存储刮削结果
func (s *scraper) saveScrapedData(task *models.ScrapeTask, data map[string]interface{}) error {
	// 创建业务数据
	businessData := &models.BusinessData{
		Module:       task.Module,
		Description:  fmt.Sprintf("刮削数据 - %s", task.DataPath),
		CustomFields: data,
		FilePath:     task.DataPath,
		BaseModel: models.BaseModel{
			CreatedBy: task.CreatedBy,
			UpdatedBy: task.CreatedBy,
		},
	}

	// 保存到动态集合
	err := s.storage.CreateBusinessData(businessData)
	if err != nil {
		return fmt.Errorf("存储业务数据失败: %v", err)
	}

	log.Info().
		Str("task_id", task.ID.Hex()).
		Str("module", task.Module).
		Str("data_id", businessData.ID.Hex()).
		Msg("刮削结果已存储")

	return nil
}
