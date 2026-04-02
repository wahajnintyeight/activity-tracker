package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"activity-tracker/internal/config"
	"activity-tracker/internal/cricket"
	"activity-tracker/internal/service"

	svc "github.com/kardianos/service"
)

func main() {
	// Parse command line arguments
	var trackerType string
	var teamScorePositionFlag string
	var gameType string
	if len(os.Args) > 1 {
		for i, arg := range os.Args {
			if arg == "--type" && i+1 < len(os.Args) {
				trackerType = os.Args[i+1]
			}
			if arg == "--team-score-position" && i+1 < len(os.Args) {
				teamScorePositionFlag = os.Args[i+1]
			}
			if arg == "--game-type" && i+1 < len(os.Args) {
				gameType = os.Args[i+1]
			}
		}
	}

	// Handle cricket tracker mode
	if trackerType == "cricket-tracker" {
		runCricketTracker(teamScorePositionFlag, gameType)
		return
	}

	// If no arguments, run in background mode (activity tracker)
	if len(os.Args) == 1 {
		// Run as a proper background service programmatically
		svcConfig := &svc.Config{
			Name:        "ScreenshotService",
			DisplayName: "Screenshot Capture Service",
			Description: "Captures and uploads screenshots periodically",
			Option: svc.KeyValue{
				"StartType": "automatic",
			},
		}

		prg := &serviceProgram{
			svcConfig: svcConfig,
		}

		s, err := svc.New(prg, svcConfig)
		if err != nil {
			log.Fatal(err)
		}

		// Run the service programmatically (not as installed Windows service)
		// This will run in background without console
		err = prg.Start(s)
		if err != nil {
			log.Fatal(err)
		}

		// Handle graceful shutdown
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

		// Wait for shutdown signal
		<-sigChan
		log.Println("Shutdown signal received, stopping service...")
		if err := prg.Stop(s); err != nil {
			log.Printf("Error stopping service: %v", err)
		}
		return
	}

	// Service configuration
	svcConfig := &svc.Config{
		Name:        "ScreenshotService",
		DisplayName: "Screenshot Capture Service",
		Description: "Captures and uploads screenshots periodically",
		Option: svc.KeyValue{
			"StartType": "automatic",
		},
	}

	// Create service program wrapper
	prg := &serviceProgram{
		svcConfig: svcConfig,
	}

	s, err := svc.New(prg, svcConfig)
	if err != nil {
		log.Fatal(err)
	}

	// Handle commands
	cmd := os.Args[1]
	switch cmd {
	case "run":
		cfg := config.Load()
		deviceName, _ := os.Hostname()
		fullService, err := service.New(cfg, deviceName)
		if err != nil {
			log.Fatal(err)
		}
		fullService.RunManually()
	case "install":
		err = s.Install()
		if err == nil {
			log.Println("Service installed successfully")
		}
	case "uninstall":
		err = s.Uninstall()
		if err == nil {
			log.Println("Service uninstalled successfully")
		}
	case "start":
		err = s.Start()
		if err == nil {
			log.Println("Service started successfully")
		}
	case "stop":
		err = s.Stop()
		if err == nil {
			log.Println("Service stopped successfully")
		}
	case "restart":
		err = s.Restart()
		if err == nil {
			log.Println("Service restarted successfully")
		}
	default:
		log.Fatalf("Unknown command: %s\nAvailable: install, uninstall, start, stop, restart, run, --type cricket-tracker", cmd)
	}
	if err != nil {
		log.Fatal(err)
	}
}

// serviceProgram implements svc.Interface
type serviceProgram struct {
	svcConfig   *svc.Config
	fullService *service.ScreenshotService
}

func (p *serviceProgram) Start(s svc.Service) error {
	// Initialization happens in a goroutine to not block service start
	go func() {
		cfg := config.Load()
		deviceName, _ := os.Hostname()
		fullService, err := service.New(cfg, deviceName)
		if err != nil {
			// Try to log via service logger if available
			if logger, err := s.Logger(nil); err == nil {
				logger.Errorf("Failed to initialize service: %v", err)
			}
			return
		}
		p.fullService = fullService

		// Setup logger if available
		if logger, err := s.Logger(nil); err == nil {
			p.fullService.SetLogger(logger)
		}

		p.fullService.Start(s)
	}()
	return nil
}

func (p *serviceProgram) Stop(s svc.Service) error {
	if p.fullService != nil {
		return p.fullService.Stop(s)
	}
	return nil
}

func runCricketTracker(teamScorePositionFlag string, gameType string) {
	log.Println("Starting Cricket Tracker...")

	cfg := config.LoadCricketConfig()
	if gameType != "" {
		cfg.GameType = gameType
	}

	// CLI flag overrides the env var value
	teamScorePosition := cfg.TeamScorePosition
	if teamScorePositionFlag != "" {
		teamScorePosition = teamScorePositionFlag
	}
	if teamScorePosition == "" {
		teamScorePosition = "left"
	}
	log.Printf("Team score position: %s", teamScorePosition)

	trackerConfig := &cricket.CricketTrackerConfig{
		RabbitMQURL:        cfg.RabbitMQURL,
		RabbitMQExchange:   cfg.RabbitMQExchange,
		RabbitMQRoutingKey: cfg.RabbitMQRoutingKey,
		DiscordAppID:       cfg.DiscordAppID,
		Interval:           cfg.Interval,
		ScoreboardX:        cfg.ScoreboardX,
		ScoreboardY:        cfg.ScoreboardY,
		ScoreboardWidth:    cfg.ScoreboardWidth,
		ScoreboardHeight:   cfg.ScoreboardHeight,
		ProcessNames:       cfg.ProcessNames,
		UseLLMOCR:          cfg.UseLLMOCR,
		DebugZones:         cfg.DebugZones,
		GameType:           cfg.GameType,
		TeamScorePosition:  teamScorePosition,
	}

	tracker, err := cricket.NewCricketTracker(trackerConfig)
	if err != nil {
		log.Fatalf("Failed to create cricket tracker: %v", err)
	}

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Shutdown signal received, stopping cricket tracker...")
		if err := tracker.Stop(); err != nil {
			log.Printf("Error stopping tracker: %v", err)
		}
		os.Exit(0)
	}()

	// Start tracking
	if err := tracker.Start(); err != nil {
		log.Fatalf("Cricket tracker error: %v", err)
	}
}
