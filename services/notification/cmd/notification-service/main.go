package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"notification-service/internal/config"
	"notification-service/internal/events"
	"notification-service/internal/service"
)

func main() {
	// Загрузка конфигурации
	cfg := config.Configuration{
		RabbitMQ: config.RabbitMQ{
			URL:       "amqp://guest:guest@rabbitmq:5672/",
			QueueName: "payments",
		},
	}

	// Инициализация RabbitMQ
	listener, err := events.NewRabbitMQListener(cfg.RabbitMQ.URL, cfg.RabbitMQ.QueueName)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer listener.Close()

	// Создание сервиса
	logger := log.New(os.Stdout, "[NOTIFICATION] ", log.LstdFlags)
	notificationService := service.NewNotificationService(listener, logger)

	// Запуск сервиса
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := notificationService.Start(ctx); err != nil {
		log.Fatalf("Failed to start notification service: %v", err)
	}

	log.Println("Notification service started. Waiting for messages...")

	// Ожидание сигнала завершения
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down notification service...")
}
