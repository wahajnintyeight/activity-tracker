package cricket

import (
	"activity-tracker/internal/queue"
	"activity-tracker/internal/window"
	"bytes"
	"fmt"
	"image"
	"image/png"
	"log"
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
	debugZones     bool     // If true, save debug images of zones
	stopChan       chan struct{}
	lastEvent      *GameEvent // Track last event to prevent duplicates
	lastEventTime  time.Time  // Track when last event was sent
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
	DebugZones         bool // If true, save debug images of zones
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
		debugZones:     config.DebugZones,
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
	events, newState := ProcessScoreWithVision(img, text, ct.matchState, ct.ocrClient, ct.debugZones)
	ct.matchState = newState

	if ct.matchState != nil {
		log.Printf("Batsmen: Left='%s', Right='%s' | Striker: '%s' (LeftIsStriker: %v)",
			ct.matchState.BatsmanLeft, ct.matchState.BatsmanRight, ct.matchState.BatsmanName, ct.matchState.IsStrikerOnLeft)
	}

	// If events detected, check duplicates and publish each
	for _, event := range events {
		if event == nil {
			continue
		}

		// Deduplicate: Check if this is the same event as the last one
		if ct.shouldPublishEvent(event) {
			log.Printf("Cricket Event Detected: %s - %s", event.Type, event.Payload)

			if err := ct.publisher.Publish(event); err != nil {
				return fmt.Errorf("failed to publish event: %w", err)
			}

			// Update last event tracking
			ct.lastEvent = event
			ct.lastEventTime = time.Now()
		} else {
			log.Printf("Duplicate event suppressed: %s", event.Type)
		}
	}

	return nil
}

// shouldPublishEvent checks if an event should be published (deduplication)
func (ct *CricketTracker) shouldPublishEvent(event *GameEvent) bool {
	// If no previous event, always publish
	if ct.lastEvent == nil {
		return true
	}

	// Check if this is a duplicate within the last 10 seconds
	timeSinceLastEvent := time.Since(ct.lastEventTime)
	if timeSinceLastEvent < 10*time.Second {
		// Same event type
		if ct.lastEvent.Type == event.Type {
			// For milestones, check if same batsman and same milestone
			if event.Type == EventTypeMilestone {
				lastBatsman := ""
				lastMilestone := ""
				currentBatsman := ""
				currentMilestone := ""

				if ct.lastEvent.MatchData != nil {
					lastBatsman = ct.lastEvent.MatchData.BatsmanName
					lastMilestone = ct.lastEvent.MatchData.MilestoneType
				}
				if event.MatchData != nil {
					currentBatsman = event.MatchData.BatsmanName
					currentMilestone = event.MatchData.MilestoneType
				}

				// Same batsman and same milestone = duplicate
				if lastBatsman == currentBatsman && lastMilestone == currentMilestone {
					return false
				}
			}

			if event.Type == EventTypeTeamMilestone {
				lastMilestone := 0
				currentMilestone := 0

				if ct.lastEvent.MatchData != nil {
					lastMilestone = ct.lastEvent.MatchData.TeamMilestoneRuns
				}
				if event.MatchData != nil {
					currentMilestone = event.MatchData.TeamMilestoneRuns
				}

				if lastMilestone == currentMilestone && currentMilestone > 0 {
					return false
				}
			}

			if event.Type == EventTypeChaseUpdate {
				lastNeedRuns := 0
				lastNeedBalls := 0
				currentNeedRuns := 0
				currentNeedBalls := 0

				if ct.lastEvent.MatchData != nil {
					lastNeedRuns = ct.lastEvent.MatchData.NeedRuns
					lastNeedBalls = ct.lastEvent.MatchData.NeedBalls
				}
				if event.MatchData != nil {
					currentNeedRuns = event.MatchData.NeedRuns
					currentNeedBalls = event.MatchData.NeedBalls
				}

				if lastNeedRuns == currentNeedRuns && lastNeedBalls == currentNeedBalls && currentNeedRuns > 0 {
					return false
				}
			}

			// For boundaries (4s and 6s), check if same score
			if event.Type == EventTypeBoundaryFour || event.Type == EventTypeBoundarySix {
				lastScore := 0
				currentScore := 0

				if ct.lastEvent.MatchData != nil {
					lastScore = ct.lastEvent.MatchData.TotalRuns
				}
				if event.MatchData != nil {
					currentScore = event.MatchData.TotalRuns
				}

				// Same score = duplicate
				if lastScore == currentScore {
					return false
				}
			}

			// For wickets, check if same wicket count
			if event.Type == EventTypeWicket || event.Type == EventTypeBatsmanDepart {
				lastWickets := 0
				currentWickets := 0

				if ct.lastEvent.MatchData != nil {
					lastWickets = ct.lastEvent.MatchData.Wickets
				}
				if event.MatchData != nil {
					currentWickets = event.MatchData.Wickets
				}

				// Same wicket count = duplicate
				if lastWickets == currentWickets {
					return false
				}
			}

			// For batsman/bowler arrivals, check if same player
			if event.Type == EventTypeBatsmanArrive {
				lastBatsman := ""
				currentBatsman := ""

				if ct.lastEvent.MatchData != nil {
					lastBatsman = ct.lastEvent.MatchData.BatsmanName
				}
				if event.MatchData != nil {
					currentBatsman = event.MatchData.BatsmanName
				}

				// Same batsman = duplicate
				if lastBatsman == currentBatsman {
					return false
				}
			}

			if event.Type == EventTypeBowlerArrive {
				lastBowler := ""
				currentBowler := ""

				if ct.lastEvent.MatchData != nil {
					lastBowler = ct.lastEvent.MatchData.BowlerName
				}
				if event.MatchData != nil {
					currentBowler = event.MatchData.BowlerName
				}

				// Same bowler = duplicate
				if lastBowler == currentBowler {
					return false
				}
			}
		}
	}

	// Not a duplicate, publish it
	return true
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
