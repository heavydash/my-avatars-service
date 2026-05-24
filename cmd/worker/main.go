package main

import (
	"context"
	"fmt"
	"github.com/heavydash/my-avatars-service/internal/config"
	"github.com/heavydash/my-avatars-service/internal/events"
	"github.com/heavydash/my-avatars-service/internal/pkg/logger"
	"github.com/heavydash/my-avatars-service/internal/repository"
	"github.com/heavydash/my-avatars-service/internal/repository/postgres"
	minio2 "github.com/heavydash/my-avatars-service/internal/storage/minio"
	"github.com/heavydash/my-avatars-service/internal/worker"
	"go.uber.org/zap"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	fmt.Println("Starting GophProfile Worker...")

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

	log.Info("Worker starting...")

	// Подключение к БД
	dbPool, err := postgres.New(context.Background(), cfg)
	if err != nil {
		log.Error("Failed to connect to database", zap.Error(err))
		os.Exit(1)
	}
	defer dbPool.Close()

	// Подключение к RabbitMQ
	rabbitMQ, err := events.NewRabbitMQ("amqp://guest:guest@localhost:5672/")
	if err != nil {
		log.Error("Failed to connect to RabbitMQ", zap.Error(err))
		os.Exit(1)
	}
	defer rabbitMQ.Close()

	// Инициализация репозитория и MinIO для Worker
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

	// Запуск Worker
	w := worker.NewWorker(dbPool.Pool, rabbitMQ.Channel(), log, avatarRepo, fileStorage)

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := w.Start(); err != nil {
			log.Error("Worker failed", zap.Error(err))
		}
	}()

	log.Info("Worker is running...")

	<-quit
	log.Info("Shutting down worker...")
	w.Stop()
	log.Info("Worker exited gracefully")
}
