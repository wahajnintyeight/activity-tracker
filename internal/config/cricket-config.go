package config

import (
	"activity-tracker/internal/cricket"
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
	DiscordAppID       string
	ScoreboardX        int
	ScoreboardY        int
	ScoreboardWidth    int
	ScoreboardHeight   int
	ProcessNames       []string
	UseLLMOCR          bool             // If true, send images to queue for LLM analysis instead of local OCR
	DebugZones         bool             // If true, save debug images of zones
	GameType           cricket.GameType // "c24" or "c26" — selects HUD zone coordinates
	TeamScorePosition  string           // "left" or "middle" — controls batsman HUD zone coordinates
	DisableEvents      bool             // If true, suppress RabbitMQ events
}

// LoadCricketConfig reads cricket tracker configuration from environment
func LoadCricketConfig(gameTypeOverride string) *CricketConfig {
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
	discordAppID := getEnv("DISCORD_APP_ID", "")

	// OCR mode: local (Windows Native) or LLM (server-side)
	useLLMOCR := getEnv("CRICKET_USE_LLM_OCR", "false") == "true"

	// Debug mode for vision zones
	debugZones := getEnv("CRICKET_DEBUG_ZONES", "false") == "true"

	// Game type: "c24" or "c26"
	gameTypeStr := getEnv("CRICKET_GAME_TYPE", string(cricket.GameTypeC24))
	if gameTypeOverride != "" {
		gameTypeStr = gameTypeOverride
	}
	gameType := cricket.GameType(gameTypeStr)

	// Process names to monitor
	defaultProcesses := "Cricket24.exe, cricket.exe, Cricket 24.exe, Cricket24-Win64-Shipping.exe, Cricket26.exe, Cricket 26.exe, Cricket26-Win64-Shipping.exe"
	processNamesStr := getEnv("CRICKET_PROCESS_NAMES", defaultProcesses)

	// Ensure game-specific shipping binaries are included if not in .env
	if gameType == cricket.GameTypeC26 && !strings.Contains(processNamesStr, "Cricket26") {
		processNamesStr += ", Cricket26.exe, Cricket 26.exe, Cricket26-Win64-Shipping.exe"
	} else if gameType == cricket.GameTypeC24 && !strings.Contains(processNamesStr, "Cricket24") {
		processNamesStr += ", Cricket24.exe, Cricket 24.exe, Cricket24-Win64-Shipping.exe"
	}
	processNames := strings.Split(processNamesStr, ",")
	for i := range processNames {
		processNames[i] = strings.TrimSpace(processNames[i])
	}

	// Get game-specific prefix (e.g., "c24" -> "24_")
	prefix := ""
	if strings.HasPrefix(string(gameType), "c") {
		prefix = string(gameType)[1:] + "_"
	}

	// Scoreboard coordinates (default for 1920x1080, bottom-left)
	// Try game-specific variables first, then fallback to general ones
	scoreboardX := getEnvInt(prefix+"CRICKET_SCOREBOARD_X", getEnvInt("CRICKET_SCOREBOARD_X", 20))
	scoreboardY := getEnvInt(prefix+"CRICKET_SCOREBOARD_Y", getEnvInt("CRICKET_SCOREBOARD_Y", 950))
	scoreboardWidth := getEnvInt(prefix+"CRICKET_SCOREBOARD_WIDTH", getEnvInt("CRICKET_SCOREBOARD_WIDTH", 300))
	scoreboardHeight := getEnvInt(prefix+"CRICKET_SCOREBOARD_HEIGHT", getEnvInt("CRICKET_SCOREBOARD_HEIGHT", 80))

	// Team score panel position: "left" or "middle"
	teamScorePosition := getEnv("CRICKET_TEAM_SCORE_POSITION", "left")

	// Disable events toggle (suppresses RabbitMQ, keeps Discord Rich Presence)
	disableEvents := getEnv("CRICKET_DISABLE_EVENTS", "false") == "true"

	return &CricketConfig{
		Interval:           interval,
		RabbitMQURL:        rabbitmqURL,
		RabbitMQExchange:   rabbitmqExchange,
		RabbitMQRoutingKey: rabbitmqRoutingKey,
		DiscordAppID:       discordAppID,
		ScoreboardX:        scoreboardX,
		ScoreboardY:        scoreboardY,
		ScoreboardWidth:    scoreboardWidth,
		ScoreboardHeight:   scoreboardHeight,
		ProcessNames:       processNames,
		UseLLMOCR:          useLLMOCR,
		DebugZones:         debugZones,
		GameType:           gameType,
		TeamScorePosition:  teamScorePosition,
		DisableEvents:      disableEvents,
	}
}
