package cricket

import (
	"activity-tracker/internal/queue"
	"activity-tracker/internal/window"
	"bytes"
	"fmt"
	"image"
	"image/png"
	"log"
	"os"
	"strings"
	"time"
)

type CricketTracker struct {
	ocrClient      OCRClient
	publisher      *queue.RabbitMQPublisher
	matchState     *MatchState
	scoreboardRect image.Rectangle
	interval       time.Duration
	processNames   []string // Cricket game process names to monitor
	useLLMOCR      bool     // If true, send images to queue instead of doing local OCR
	stopChan       chan struct{}
}

type CricketTrackerConfig struct {
	RabbitMQURL        string
	RabbitMQExchange   string
	RabbitMQRoutingKey string
	Interval           time.Duration
	ScoreboardX        int
	ScoreboardY        int
	ScoreboardWidth    int
	ScoreboardHeight   int
	ProcessNames       []string
	UseLLMOCR          bool // If true, send images to queue for LLM analysis
}

// NewCricketTracker creates a new cricket tracking service
func NewCricketTracker(config *CricketTrackerConfig) (*CricketTracker, error) {
	// Initialize OCR client (only if not using LLM OCR)
	var ocrClient OCRClient
	if !config.UseLLMOCR {
		ocrClient = NewOCRClient()
	}

	// Initialize RabbitMQ publisher
	publisher, err := queue.NewRabbitMQPublisher(
		config.RabbitMQURL,
		config.RabbitMQExchange,
		config.RabbitMQRoutingKey,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create RabbitMQ publisher: %w", err)
	}

	// Set default process names if not provided
	processNames := config.ProcessNames
	if len(processNames) == 0 {
		processNames = []string{"Cricket24.exe", "cricket.exe", "Cricket 24.exe"}
	}

	mode := "Local OCR (Windows Native)"
	if config.UseLLMOCR {
		mode = "LLM OCR (Server-side)"
	}
	log.Printf("Cricket Tracker Mode: %s", mode)

	return &CricketTracker{
		ocrClient:      ocrClient,
		publisher:      publisher,
		matchState:     nil,
		scoreboardRect: GetScoreboardRect(config.ScoreboardX, config.ScoreboardY, config.ScoreboardWidth, config.ScoreboardHeight),
		interval:       config.Interval,
		processNames:   processNames,
		useLLMOCR:      config.UseLLMOCR,
		stopChan:       make(chan struct{}),
	}, nil
}

// Start begins the cricket tracking loop
func (ct *CricketTracker) Start() error {
	log.Println("Cricket Tracker started")
	log.Printf("Monitoring processes: %v", ct.processNames)
	log.Printf("Scoreboard area: %v", ct.scoreboardRect)
	log.Printf("Scan interval: %v", ct.interval)

	ticker := time.NewTicker(ct.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := ct.processFrame(); err != nil {
				log.Printf("Error processing frame: %v", err)
			}
		case <-ct.stopChan:
			log.Println("Cricket Tracker stopped")
			return nil
		}
	}
}

// Stop gracefully stops the tracker
func (ct *CricketTracker) Stop() error {
	close(ct.stopChan)

	if ct.ocrClient != nil {
		ct.ocrClient.Close()
	}

	if ct.publisher != nil {
		ct.publisher.Close()
	}

	return nil
}

// processFrame captures and analyzes a single frame
func (ct *CricketTracker) processFrame() error {
	// Check if cricket game is in foreground
	activeWin, err := window.GetActive()
	if err != nil {
		return fmt.Errorf("failed to get active window: %w", err)
	}

	// Check if active process matches any cricket game
	isCricketActive := true
	for _, procName := range ct.processNames {
		if strings.EqualFold(activeWin.ProcessName, procName) {
			isCricketActive = true
			break
		}
	}

	if !isCricketActive {
		// Cricket not active, skip processing
		return nil
	}

	// Capture scoreboard area
	img, err := CaptureScoreboardArea(ct.scoreboardRect)
	if err != nil {
		return fmt.Errorf("failed to capture scoreboard: %w", err)
	}

	// Save debug image (helps with coordinate adjustment)
	debugDir := "debug"
	if _, err := os.Stat(debugDir); os.IsNotExist(err) {
		os.Mkdir(debugDir, 0755)
	}

	debugPath := fmt.Sprintf("%s/cricket-debug-%d.png", debugDir, time.Now().Unix())
	if debugFile, err := os.Create(debugPath); err == nil {
		png.Encode(debugFile, img)
		debugFile.Close()
		log.Printf("Debug image saved: %s", debugPath)
	}

	// Two modes: Local OCR or LLM OCR
	if ct.useLLMOCR {
		// LLM OCR Mode: Send image to queue for server-side analysis
		return ct.processFrameLLM(img)
	}

	// Local OCR Mode: Extract text locally and detect events
	return ct.processFrameLocal(img)
}

// processFrameLocal handles local OCR and event detection
func (ct *CricketTracker) processFrameLocal(img *image.RGBA) error {
	// Extract text via OCR
	text, err := ct.ocrClient.ExtractText(img)
	if err != nil {
		return fmt.Errorf("OCR failed: %w", err)
	}

	log.Printf("OCR Text: '%s'", text)

	if text == "" {
		log.Println("OCR returned empty - check coordinates in .env file")
		return nil
	}

	// Process score and detect events
	event, newState := ProcessScore(text, ct.matchState)
	ct.matchState = newState

	// If event detected, publish to queue
	if event != nil {
		log.Printf("Cricket Event Detected: %s - %s", event.Type, event.Payload)

		if err := ct.publisher.Publish(event); err != nil {
			return fmt.Errorf("failed to publish event: %w", err)
		}
	}

	return nil
}

// processFrameLLM sends the scoreboard image to queue for LLM analysis
func (ct *CricketTracker) processFrameLLM(img *image.RGBA) error {
	// Encode image to PNG bytes
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return fmt.Errorf("failed to encode image: %w", err)
	}

	// Create a payload with the image data
	payload := &CricketImagePayload{
		Type:      "CRICKET_SCOREBOARD",
		Timestamp: time.Now().Unix(),
		ImageData: buf.Bytes(),
	}

	log.Printf("Sending scoreboard image to LLM for analysis (%d bytes)", len(payload.ImageData))

	if err := ct.publisher.Publish(payload); err != nil {
		return fmt.Errorf("failed to publish image: %w", err)
	}

	return nil
}
