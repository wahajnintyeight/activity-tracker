package config

import (
	"github.com/joho/godotenv"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

type Config struct {
	Interval           time.Duration
	Quality            int
	RabbitMQURL        string
	RabbitMQExchange   string
	RabbitMQRoutingKey string
}

// Load reads configuration from environment variables or uses defaults
func Load() *Config {
	// Try to load .env from executable directory
	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		godotenv.Load(filepath.Join(exeDir, ".env"))
	}

	// Also try current directory
	godotenv.Load()

	interval := getEnvDuration("SCREENSHOT_INTERVAL", 5*time.Minute)
	quality := getEnvInt("JPEG_QUALITY", 50)
	rabbitmqURL := getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/")
	rabbitmqExchange := getEnv("RABBITMQ_EXCHANGE", "worker-service-exchange")
	rabbitmqRoutingKey := getEnv("RABBITMQ_ROUTING_KEY", "process-screenshot")
	print(rabbitmqURL)
	return &Config{
		Interval:           interval,
		Quality:            quality,
		RabbitMQURL:        rabbitmqURL,
		RabbitMQExchange:   rabbitmqExchange,
		RabbitMQRoutingKey: rabbitmqRoutingKey,
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
