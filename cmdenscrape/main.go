package main

import (
	"fmt"
	"math/rand"
	"time"

	"datacenter/internal/models"
	"datacenter/internal/storage"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func main() {
	fmt.Println("正在连接到业务数据库...")
	storage, err := storage.NewMongoDBStorage(
		"mongodb://localhost:27017",
		"datacenter",
	)
	if err != nil {
		panic("连接数据库失败: " + err.Error())
	}

	fmt.Println("正在生成测试刮削任务数据...")

	// 生成测试模块
	modules := []string{"movie", "music", "book", "game", "product"}

	// 生成不同状态的刮削任务
	for i := 0; i < 20; i++ {
		module := modules[rand.Intn(len(modules))]
		status := getRandomStatus()
		task := createTestScrapeTask(module, status, i)
		
		if err := storage.CreateScrapeTask(task); err != nil {
			fmt.Printf("创建刮削任务失败: %v\n", err)
			continue
		}
		
		fmt.Printf("创建刮削任务: %s - %s (状态: %s)\n", module, task.ID.Hex(), task.Status)
	}

	fmt.Println("测试数据生成完成!")
}

func getRandomStatus() models.ScrapeTaskStatus {
	statuses := []models.ScrapeTaskStatus{
		models.ScrapeTaskStatusScraping,
		models.ScrapeTaskStatusSuccess,
		models.ScrapeTaskStatusFailed,
	}
	return statuses[rand.Intn(len(statuses))]
}

func createTestScrapeTask(module string, status models.ScrapeTaskStatus, index int) *models.ScrapeTask {
	now := time.Now()
	startedAt := now.Add(-time.Duration(rand.Intn(72)) * time.Hour)
	
	var completedAt *time.Time
	if status != models.ScrapeTaskStatusScraping {
		completed := startedAt.Add(time.Duration(1+rand.Intn(24)) * time.Hour)
		completedAt = &completed
	}

	task := &models.ScrapeTask{
		BaseModel: models.BaseModel{
			ID:        primitive.NewObjectID(),
			CreatedBy: "system",
			CreatedAt: now,
			UpdatedBy: "system",
			UpdatedAt: now,
		},
		Module:      module,
		DataPath:    fmt.Sprintf("/data/%s/%d", module, index),
		ScraperPath: fmt.Sprintf("/scrapers/%s_scraper.py", module),
		Status:      status,
		StartedAt:   &startedAt,
		CompletedAt: completedAt,
	}

	switch status {
	case models.ScrapeTaskStatusSuccess:
		task.Result = map[string]interface{}{
			"items_scraped": 100 + rand.Intn(900),
			"duration":      (completedAt.Sub(*task.StartedAt)).String(),
			"success":       true,
			"details": map[string]interface{}{
				"source":     fmt.Sprintf("https://example.com/%s", module),
				"categories": []string{"category1", "category2"},
				"processed":  true,
			},
		}
	case models.ScrapeTaskStatusFailed:
		task.ErrorMessage = fmt.Sprintf("Error scraping %s data: connection timeout", module)
		task.Result = map[string]interface{}{
			"items_scraped": 0,
			"duration":      (completedAt.Sub(*task.StartedAt)).String(),
			"success":       false,
		}
	case models.ScrapeTaskStatusScraping:
		task.Result = map[string]interface{}{
			"items_scraped": 50 + rand.Intn(150),
			"duration":      (time.Since(*task.StartedAt)).String(),
			"status":        "in_progress",
		}
	}

	return task
}
