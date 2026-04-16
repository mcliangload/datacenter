package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"datacenter/internal/api"
	"datacenter/internal/auth"
	"datacenter/internal/logger"
	"datacenter/internal/storage"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// 加载环境变量
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	// 初始化日志系统
	logger.Init(
		getEnv("LOG_LEVEL", "info"),
		getEnv("LOG_FILE", "logs/app.log"),
		getEnvAsInt("LOG_MAX_SIZE", 100),
		getEnvAsInt("LOG_MAX_BACKUPS", 5),
		getEnvAsInt("LOG_MAX_AGE", 30),
	)

	// 初始化存储层（业务数据）
	businessStorage, err := storage.NewMongoDBStorage(
		getEnv("MONGODB_URI", "mongodb://localhost:27017"),
		getEnv("MONGODB_DATABASE", "datacenter"),
	)
	if err != nil {
		logger.Error("Failed to initialize business storage: %v", err)
		log.Fatalf("Failed to initialize business storage: %v", err)
	}

	// 初始化用户存储（RBAC专用）
	userStorage, err := storage.NewMongoDBStorage(
		getEnv("MONGODB_URI", "mongodb://localhost:27017"),
		"user", // 专属user数据库
	)
	if err != nil {
		logger.Error("Failed to initialize user storage: %v", err)
		log.Fatalf("Failed to initialize user storage: %v", err)
	}

	// 初始化默认权限和角色
	if err := userStorage.InitDefaultData(); err != nil {
		logger.Error("Failed to initialize default data: %v", err)
		log.Fatalf("Failed to initialize default data: %v", err)
	}

	// 初始化JWT服务
	jwtService := auth.NewJWTService(
		getEnv("JWT_SECRET", "your-secret-key"),
		time.Duration(getEnvAsInt("JWT_EXPIRATION", 24))*time.Hour,
		time.Duration(getEnvAsInt("JWT_REFRESH_EXPIRATION", 720))*time.Hour,
	)

	// 初始化API处理器
	handler := api.NewHandler(businessStorage, userStorage)

	// 初始化Gin引擎
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	// 添加中间件
	router.Use(logger.LoggerMiddleware())
	router.Use(gin.Recovery())

	// 注册路由
	handler.RegisterRoutes(router, jwtService)

	// 配置服务器
	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%s", getEnv("SERVER_HOST", "0.0.0.0"), getEnv("SERVER_PORT", "8080")),
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	// 启动服务器
	go func() {
		logger.Info("Server starting on %s", server.Addr)
		if err := server.ListenAndServeTLS(
			getEnv("TLS_CERT", "cert.pem"),
			getEnv("TLS_KEY", "key.pem"),
		); err != nil && err != http.ErrServerClosed {
			logger.Error("Failed to start server: %v", err)
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("Shutting down server...")

	// 优雅关闭
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown: %v", err)
	}

	logger.Info("Server exiting")
}

// getEnv 获取环境变量，如果不存在则返回默认值
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// getEnvAsInt 获取环境变量并转换为整数，如果不存在或转换失败则返回默认值
func getEnvAsInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
