package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"datacenter/internal/api"
	"datacenter/internal/auth"
	"datacenter/internal/logger"
	"datacenter/internal/scraper"
	"datacenter/internal/storage"
	"datacenter/pkg/rbac"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// 尝试加载项目根目录的.env文件
	if err := godotenv.Load("D:\\gocode\\datacenter\\.env"); err != nil {
		panic("No .env file found, using system environment variables")
	}

	fmt.Println("1. 环境变量加载完成")

	logger.Init(
		getEnv("LOG_LEVEL", "info"),
		getEnv("LOG_HTTP_FILE", "logs/http.log"),
		getEnvAsInt("LOG_MAX_SIZE", 100),
		getEnvAsInt("LOG_MAX_BACKUPS", 5),
		getEnvAsInt("LOG_MAX_AGE", 30),
	)

	fmt.Println("2. 日志初始化完成")

	businessStorage, err := storage.NewMongoDBStorage(
		getEnv("MONGODB_URI", "mongodb://localhost:27017"),
		getEnv("MONGODB_DATABASE", "datacenter"),
	)
	if err != nil {
		panic("Failed to initialize business storage: " + err.Error())
	}
	fmt.Println("3. 业务存储初始化完成")

	rbacStorage, err := storage.NewRBACMongoDBStorage(
		getEnv("MONGODB_RBAC_URI", "mongodb://localhost:27017"),
		getEnv("MONGODB_RBAC_DATABASE", "rbac"),
	)
	if err != nil {
		panic("Failed to initialize RBAC storage: " + err.Error())
	}
	fmt.Println("4. RBAC存储初始化完成")

	if err := rbacStorage.InitDefaultData(); err != nil {
		panic("Failed to initialize default data: " + err.Error())
	}
	fmt.Println("5. 默认数据初始化完成")

	scraperSystem := scraper.NewScraper(businessStorage, getEnvAsInt("SCRAPER_WORKERS", 4))
	if err := scraperSystem.Start(); err != nil {
		panic("Failed to start scraper system: " + err.Error())
	}
	defer scraperSystem.Stop()
	fmt.Println("6. 刮削系统启动完成")

	jwtService := auth.NewJWTService(
		getEnv("JWT_SECRET", "your-secret-key"),
		time.Duration(getEnvAsInt("JWT_EXPIRATION", 24))*time.Hour,
		time.Duration(getEnvAsInt("JWT_REFRESH_EXPIRATION", 720))*time.Hour,
	)
	fmt.Println("7. JWT服务初始化完成")

	rbacService := rbac.NewService(rbacStorage)
	fmt.Println("7.5 RBAC服务初始化完成")

	handler := api.NewHandler(businessStorage, rbacStorage, scraperSystem, jwtService, rbacService)
	fmt.Println("8. API处理器初始化完成")

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	// 添加CORS中间件
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	router.Use(logger.LoggerMiddleware())
	router.Use(gin.Recovery())

	handler.RegisterRoutes(router)
	fmt.Println("9. 路由注册完成")

	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%s", getEnv("SERVER_HOST", "0.0.0.0"), getEnv("SERVER_PORT", "8080")),
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	fmt.Println("10. HTTP服务器创建完成，地址:", server.Addr)

	go func() {
		logger.Info("Server starting on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("服务器启动失败: %v", err)
			panic("Failed to start server: " + err.Error())
		}
	}()
	fmt.Println("11. 服务器启动中...")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown: %v", err)
	}

	logger.Info("Server exiting")
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
