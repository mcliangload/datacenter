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

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		panic("No .env file found, using system environment variables")
	}

	logger.Init(
		getEnv("LOG_LEVEL", "info"),
		getEnv("LOG_HTTP_FILE", "logs/http.log"),
		getEnvAsInt("LOG_MAX_SIZE", 100),
		getEnvAsInt("LOG_MAX_BACKUPS", 5),
		getEnvAsInt("LOG_MAX_AGE", 30),
	)

	businessStorage, err := storage.NewMongoDBStorage(
		getEnv("MONGODB_URI", "mongodb://localhost:27017"),
		getEnv("MONGODB_DATABASE", "datacenter"),
	)
	if err != nil {
		panic("Failed to initialize business storage: " + err.Error())
	}

	rbacStorage, err := storage.NewRBACMongoDBStorage(
		getEnv("MONGODB_RBAC_URI", "mongodb://localhost:27017"),
		getEnv("MONGODB_RBAC_DATABASE", "rbac"),
	)
	if err != nil {
		panic("Failed to initialize RBAC storage: " + err.Error())
	}

	if err := rbacStorage.InitDefaultData(); err != nil {
		panic("Failed to initialize default data: " + err.Error())
	}

	// 创建刮削系统
	scraperSystem := scraper.NewScraper(businessStorage, getEnvAsInt("SCRAPER_WORKERS", 4))
	if err := scraperSystem.Start(); err != nil {
		panic("Failed to start scraper system: " + err.Error())
	}
	defer scraperSystem.Stop()

	jwtService := auth.NewJWTService(
		getEnv("JWT_SECRET", "your-secret-key"),
		time.Duration(getEnvAsInt("JWT_EXPIRATION", 24))*time.Hour,
		time.Duration(getEnvAsInt("JWT_REFRESH_EXPIRATION", 720))*time.Hour,
	)

	handler := api.NewHandler(businessStorage, rbacStorage, scraperSystem)

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	router.Use(logger.LoggerMiddleware())
	router.Use(gin.Recovery())

	handler.RegisterRoutes(router, jwtService)

	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%s", getEnv("SERVER_HOST", "0.0.0.0"), getEnv("SERVER_PORT", "8080")),
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		logger.Info("Server starting on %s", server.Addr)
		if err := server.ListenAndServeTLS(
			getEnv("TLS_CERT", "cert.pem"),
			getEnv("TLS_KEY", "key.pem"),
		); err != nil && err != http.ErrServerClosed {
			panic("Failed to start server: " + err.Error())
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("Shutting down server...")

	// 停止刮削系统
	logger.Info("Stopping scraper system...")
	scraperSystem.Stop()

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
