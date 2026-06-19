package config

import (
	"log/slog"
	"os"

	"github.com/joho/godotenv"
)

// Config holds all application configuration loaded from environment variables.
// .env is loaded first (local dev), then real env vars take precedence.
type Config struct {
	// Server
	Addr        string
	GinMode     string
	Env         string // dev | prod
	APIKey      string
	MetricsAddr string

	// Database
	DatabaseURL string

	// Kafka
	KafkaBroker string

	// Logging
	LogLevel  string // debug | info | warn | error
	LogFormat string // json | text
}

// Load reads the .env file (if present) then maps environment variables
// into a Config. Real env vars always override .env values.
func Load() *Config {
	_ = godotenv.Load() // silently ignore missing .env

	return &Config{
		Addr:        envOr("ADDR", ":8081"),
		GinMode:     envOr("GIN_MODE", "release"),
		Env:         envOr("ENV", "dev"),
		APIKey:      os.Getenv("API_KEY"),
		MetricsAddr: envOr("METRICS_ADDR", ":9091"),
		DatabaseURL: envOr("DATABASE_URL", "postgres://fizz:fizz@localhost:5433/fizz?sslmode=disable"),
		KafkaBroker: envOr("KAFKA_BROKER", "localhost:9094"),
		LogLevel:    envOr("LOG_LEVEL", "info"),
		LogFormat:   envOr("LOG_FORMAT", "json"),
	}
}

// SetupLogger configures the global slog logger based on the config.
// JSON handler for production, text handler for local development.
func (c *Config) SetupLogger() {
	var level slog.Level
	switch c.LogLevel {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{Level: level}

	var handler slog.Handler
	if c.LogFormat == "text" {
		handler = slog.NewTextHandler(os.Stdout, opts)
	} else {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}

	slog.SetDefault(slog.New(handler))
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
