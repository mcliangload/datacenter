package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"go.uber.org/zap"

	"datacenter/internal/config"
	"datacenter/internal/database"
	"datacenter/internal/logger"
	"datacenter/internal/scrape"
)

// 刮削子系统独立入口：与主服务解耦，通过共享 MongoDB 任务队列协作。
// 用法：go run ./cmd/scraper -config config/config.yaml
func main() {
	var cfgPath string
	flag.StringVar(&cfgPath, "config", "config/config.yaml", "配置文件路径")
	flag.Parse()

	cfg, err := config.Load(cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "加载配置失败: %v\n", err)
		os.Exit(1)
	}
	if err := logger.Init(cfg.Log.Level, cfg.Log.Output); err != nil {
		fmt.Fprintf(os.Stderr, "初始化日志失败: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	db, err := database.Connect(cfg.Database)
	if err != nil {
		logger.L().Fatal("MongoDB 连接失败", zap.Error(err))
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = db.Close(ctx)
	}()

	if err := scrape.Run(db.DB, cfg.Scrape); err != nil {
		logger.L().Fatal("刮削子系统异常退出", zap.Error(err))
	}
	logger.L().Info("刮削子系统已退出")
}
