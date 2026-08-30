package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	"datacenter/internal/config"
	"datacenter/internal/database"
	"datacenter/internal/logger"
	"datacenter/internal/router"
)

func main() {
	var cfgPath string
	flag.StringVar(&cfgPath, "config", "config/config.yaml", "配置文件路径")
	flag.Parse()

	// 1. 加载配置
	cfg, err := config.Load(cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "加载配置失败: %v\n", err)
		os.Exit(1)
	}

	// 安全增强 P0-2：生产模式（release）禁止默认密钥与默认管理员口令，防止裸奔
	if cfg.Server.Mode == "release" {
		if cfg.JWT.Secret == "datacenter-dev-secret" {
			fmt.Fprintln(os.Stderr, "安全校验失败: 生产环境禁止使用默认 JWT secret（datacenter-dev-secret），请通过环境变量 DATACENTER_JWT_SECRET 设置强随机密钥")
			os.Exit(1)
		}
		if cfg.Bootstrap.AdminPassword == "admin123" {
			fmt.Fprintln(os.Stderr, "安全校验失败: 生产环境禁止使用默认管理员口令（admin123），请通过环境变量 DATACENTER_BOOTSTRAP_ADMIN_PASSWORD 设置强口令")
			os.Exit(1)
		}
	} else {
		if cfg.JWT.Secret == "datacenter-dev-secret" {
			fmt.Fprintln(os.Stderr, "[安全告警] 正在使用默认 JWT secret（仅限开发/测试环境）")
		}
		if cfg.Bootstrap.AdminPassword == "admin123" {
			fmt.Fprintln(os.Stderr, "[安全告警] 正在使用默认管理员口令（仅限开发/测试环境）")
		}
	}

	// 2. 初始化日志
	if err := logger.Init(cfg.Log.Level, cfg.Log.Output); err != nil {
		fmt.Fprintf(os.Stderr, "初始化日志失败: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	// 3. 连接 MongoDB
	db, err := database.Connect(cfg.Database)
	if err != nil {
		logger.L().Fatal("MongoDB 连接失败", zap.Error(err))
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = db.Close(ctx)
	}()

	// 4. 初始化索引与种子数据
	if err := database.EnsureIndexes(db.DB); err != nil {
		logger.L().Fatal("初始化 MongoDB 索引失败", zap.Error(err))
	}
	if err := database.EnsureBootstrapAdmin(db.DB, cfg.Bootstrap); err != nil {
		logger.L().Fatal("初始化默认管理员失败", zap.Error(err))
	}

	// 5. 构建路由并启动 HTTP 服务
	r := router.New(cfg, db)
	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
	}

	go func() {
		logger.L().Info("HTTP 服务启动",
			zap.Int("port", cfg.Server.Port),
			zap.String("mode", cfg.Server.Mode))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.L().Fatal("HTTP 服务异常退出", zap.Error(err))
		}
	}()

	// 6. 优雅退出
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.L().Info("收到退出信号，开始优雅关闭...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.L().Error("优雅关闭失败", zap.Error(err))
	}
	logger.L().Info("服务已退出")
}
