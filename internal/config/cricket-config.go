package config

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type CricketConfig struct {
	Interval           time.Duration
	RabbitMQURL        string
	RabbitMQExchange   string
	RabbitMQRoutingKey string
	ScoreboardX        int
	ScoreboardY        int
	ScoreboardWidth    int
	ScoreboardHeight   int
	ProcessNames       []string
	UseLLMOCR           bool   // If true, send images to queue for LLM analysis instead of local OCR
	DebugZones          bool   // If true, save debug images of zones
	GameType            string // "c24" or "c26" — selects HUD zone coordinates
	TeamScorePosition   string // "left" or "middle" — controls batsman HUD zone coordinates
}

// LoadCricketConfig reads cricket tracker configuration from environment
func LoadCricketConfig() *CricketConfig {
	// Try to load .env from executable directory
	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		godotenv.Load(filepath.Join(exeDir, ".env"))
	}

	// Also try current directory
	godotenv.Load()

	interval := getEnvDuration("CRICKET_SCAN_INTERVAL", 2*time.Second)
	rabbitmqURL := getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/")
	rabbitmqExchange := getEnv("CRICKET_RABBITMQ_EXCHANGE", "worker-service-exchange")
	rabbitmqRoutingKey := getEnv("CRICKET_RABBITMQ_ROUTING_KEY", "cricket-event")

	// Scoreboard coordinates (default for 1920x1080, bottom-left)
	scoreboardX := getEnvInt("CRICKET_SCOREBOARD_X", 20)
	scoreboardY := getEnvInt("CRICKET_SCOREBOARD_Y", 950)
	scoreboardWidth := getEnvInt("CRICKET_SCOREBOARD_WIDTH", 300)
	scoreboardHeight := getEnvInt("CRICKET_SCOREBOARD_HEIGHT", 80)

	// Process names to monitor
	processNamesStr := getEnv("CRICKET_PROCESS_NAMES", "Cricket24.exe,cricket.exe,Cricket 24.exe")
	processNames := strings.Split(processNamesStr, ",")
	for i := range processNames {
		processNames[i] = strings.TrimSpace(processNames[i])
	}

	// OCR mode: local (Windows Native) or LLM (server-side)
	useLLMOCR := getEnv("CRICKET_USE_LLM_OCR", "false") == "true"

	// Debug mode for vision zones
	debugZones := getEnv("CRICKET_DEBUG_ZONES", "false") == "true"

	// Game type: "c24" or "c26"
	gameType := getEnv("CRICKET_GAME_TYPE", "c24")

	// Team score panel position: "left" or "middle"
	teamScorePosition := getEnv("CRICKET_TEAM_SCORE_POSITION", "left")

	return &CricketConfig{
		Interval:           interval,
		RabbitMQURL:        rabbitmqURL,
		RabbitMQExchange:   rabbitmqExchange,
		RabbitMQRoutingKey: rabbitmqRoutingKey,
		ScoreboardX:        scoreboardX,
		ScoreboardY:        scoreboardY,
		ScoreboardWidth:    scoreboardWidth,
		ScoreboardHeight:   scoreboardHeight,
		ProcessNames:        processNames,
		UseLLMOCR:           useLLMOCR,
		DebugZones:          debugZones,
		GameType:            gameType,
		TeamScorePosition:   teamScorePosition,
	}
}
