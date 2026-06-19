package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/houssemlou/fizz/internal/config"
	"github.com/houssemlou/fizz/internal/kafka"
	"github.com/houssemlou/fizz/internal/producer"
	"github.com/houssemlou/fizz/internal/repository/postgres"
	"github.com/houssemlou/fizz/internal/worker"
)

func main() {
	cfg := config.Load()
	cfg.SetupLogger()

	ctx := context.Background()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("database connect failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	repo := postgres.NewRepository(pool)
	statsHandler := worker.NewStatsHandler(repo)

	c, err := kafka.NewConsumer(cfg.KafkaBroker, producer.FizzBuzzRequestsTopic, "fizzbuzz-stats", statsHandler)
	if err != nil {
		slog.Error("kafka consumer failed", "error", err)
		os.Exit(1)
	}
	defer c.Close()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	workerCtx, cancel := context.WithCancel(ctx)
	go c.Run(workerCtx)

	slog.Info("worker started", "broker", cfg.KafkaBroker, "topic", producer.FizzBuzzRequestsTopic)
	<-quit

	slog.Info("shutdown signal received")
	cancel()
	slog.Info("worker stopped")
}
