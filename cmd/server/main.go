package main

import (
	"context"
	"fmt"
	"github.com/heavydash/my-avatars-service/internal/api"
	"github.com/heavydash/my-avatars-service/internal/api/handler"
	"github.com/heavydash/my-avatars-service/internal/config"
	"github.com/heavydash/my-avatars-service/internal/pkg/logger"
	"github.com/heavydash/my-avatars-service/internal/repository"
	"github.com/heavydash/my-avatars-service/internal/repository/postgres"
	"github.com/heavydash/my-avatars-service/internal/service"
	minio2 "github.com/heavydash/my-avatars-service/internal/storage/minio"
	"go.uber.org/zap"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

var (
	buildVersion = "dev"
	buildDate    = "unknown"
	buildCommit  = "unknown"
)

func main() {
	fmt.Printf("Build version: %s\n", buildVersion)
	fmt.Printf("Build date:    %s\n", buildDate)
	fmt.Printf("Build commit:  %s\n", buildCommit)
	fmt.Println("---")

	cfg, err := config.New()
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	log, err := logger.New(cfg)
	if err != nil {
		fmt.Printf("Failed to create logger: %v\n", err)
		os.Exit(1)
	}
	defer log.Sync()

	log.Info("Starting GophProfile service...",
		zap.String("env", cfg.Server.Env),
		zap.String("port", cfg.Server.Port),
	)

	// Инициализация БД
	initCtx, initCancel := context.WithTimeout(context.Background(), cfg.DB.MigrationTimeout)
	defer initCancel()

	dbPool, err := postgres.New(initCtx, cfg)
	if err != nil {
		log.Error("Failed to create connection pool", zap.Error(err))
		os.Exit(1)
	}
	defer dbPool.Close()

	log.Info("Successfully connected to database")

	// Инициализация репозитория и storage
	avatarRepo, err := repository.NewAvatarRepository(cfg, dbPool.Pool)
	if err != nil {
		log.Error("Failed to create avatar repository", zap.Error(err))
		os.Exit(1)
	}

	fileStorage, err := minio2.NewMinIOStorage(cfg)
	if err != nil {
		log.Error("Failed to create minio storage", zap.Error(err))
		os.Exit(1)
	}

	// Сервис
	avatarService := service.NewAvatarService(avatarRepo, fileStorage)

	// Handler
	avatarHandler := handler.NewAvatarHandler(avatarService)

	log.Info("All layers initialized successfully")

	// Настройка роутера Gin
	r := api.NewRouter(avatarHandler, log)

	// HTTP сервер
	srv := &http.Server{
		Addr:    cfg.Server.Addr(),
		Handler: r,
	}

	// Запуск сервера в горутине
	go func() {
		log.Info("HTTP server starting", zap.String("address", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Server failed", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error("Server forced to shutdown", zap.Error(err))
	}

	log.Info("Server exited gracefully")
}
