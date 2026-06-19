package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/houssemlou/fizz/internal/config"
	"github.com/houssemlou/fizz/internal/domain/fizzbuzz"
	"github.com/houssemlou/fizz/internal/handler"
	"github.com/houssemlou/fizz/internal/kafka"
	appmetrics "github.com/houssemlou/fizz/internal/metrics"
	"github.com/houssemlou/fizz/internal/producer"
	"github.com/houssemlou/fizz/internal/repository/postgres"
	"github.com/houssemlou/fizz/migrations"
)

func main() {
	cfg := config.Load()
	cfg.SetupLogger()

	gin.SetMode(cfg.GinMode)

	if err := postgres.Migrate(cfg.DatabaseURL, migrations.FS); err != nil {
		slog.Error("migration failed", "error", err)
		os.Exit(1)
	}

	ctx := context.Background()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("database connect failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	p, err := kafka.NewProducer(cfg.KafkaBroker)
	if err != nil {
		slog.Error("kafka producer failed", "error", err)
		os.Exit(1)
	}
	defer p.Close()

	repo := postgres.NewRepository(pool)
	fizzbuzzService := fizzbuzz.NewService(producer.NewFizzBuzzSender(p), repo)

	metricsSrv := appmetrics.NewServer(cfg.MetricsAddr)
	go func() {
		slog.Info("metrics server starting", "addr", cfg.MetricsAddr)
		if err := metricsSrv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			slog.Error("metrics server error", "error", err)
		}
	}()

	srv := handler.New(cfg.Addr, cfg.Env, cfg.APIKey, handler.NewHandler(fizzbuzzService))

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := srv.Start(); !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-quit
	slog.Info("shutdown signal received")

	shutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	metricsSrv.Shutdown(shutCtx) //nolint:errcheck
	if err := srv.Shutdown(shutCtx); err != nil {
		slog.Error("shutdown error", "error", err)
		os.Exit(1)
	}
	slog.Info("server stopped")
}
