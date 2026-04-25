package service

import (
	"log"
	"os"
	"path/filepath"
	"time"

	"activity-tracker/internal/capture"
	"activity-tracker/internal/config"
	"activity-tracker/internal/discord"
	"activity-tracker/internal/idle"
	"activity-tracker/internal/uploader"

	svc "github.com/kardianos/service"
)

type ScreenshotService struct {
	config        *config.Config
	deviceName    string
	logger        svc.Logger
	capturer      *capture.Capturer
	uploader      *uploader.Uploader
	discordClient *discord.DiscordClient
}

func New(cfg *config.Config, deviceName string) (*ScreenshotService, error) {
	// Initialize uploader with RabbitMQ
	uploaderInstance, err := uploader.New(
		deviceName,
		cfg.RabbitMQURL,
		cfg.RabbitMQExchange,
		cfg.RabbitMQRoutingKey,
	)

	if err != nil {
		return nil, err
	}

	return &ScreenshotService{
		config:        cfg,
		deviceName:    deviceName,
		capturer:      capture.New(cfg.Quality),
		uploader:      uploaderInstance,
		discordClient: discord.NewDiscordClient(cfg.DiscordAppID),
	}, nil
}

func (s *ScreenshotService) SetLogger(logger svc.Logger) {
	s.logger = logger
}

func (s *ScreenshotService) Start(svc svc.Service) error {
	s.logger.Info("Starting screenshot service")

	// Setup file logging in background
	go s.setupFileLogging()

	// Start the main service loop
	go s.run()
	return nil
}

func (s *ScreenshotService) setupFileLogging() {
	// Get executable directory
	exePath, err := os.Executable()
	if err != nil {
		s.logger.Warningf("Could not get executable path: %v", err)
		return
	}
	exeDir := filepath.Dir(exePath)

	// Create logs directory
	logDir := filepath.Join(exeDir, "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		s.logger.Warningf("Could not create log directory: %v", err)
		return
	}

	// Set log output to file
	// log.SetOutput(f)
	log.SetFlags(log.LstdFlags)
	log.Println("File logging initialized")
}

func (s *ScreenshotService) run() {
	ticker := time.NewTicker(s.config.Interval)
	defer ticker.Stop()

	for range ticker.C {
		// Check if user is idle
		// isIdle, err := idle.IsIdle(1 * time.Minute)
		// if err != nil {
		// 	s.logger.Warningf("Could not check idle time: %v", err)
		// 	log.Printf("Could not check idle time: %v", err)
		// }

		// if isIdle {
		// 	s.logger.Info("User is idle, skipping screenshot")
		// 	log.Println("User is idle, skipping screenshot")
		// 	continue
		// }

		if err := s.captureAndSend(); err != nil {
			s.logger.Errorf("Error capturing/sending screenshot: %v", err)
			log.Printf("Error capturing/sending screenshot: %v", err)
		} else {
			log.Println("Screenshot captured and sent successfully")
		}
	}
}

func (s *ScreenshotService) captureAndSend() error {
	// Capture screenshot with metadata
	screenshot, err := s.capturer.Capture()
	if err != nil {
		return err
	}

	// Upload to API
	return s.uploader.Upload(screenshot)
}

func (s *ScreenshotService) Stop(svc svc.Service) error {
	s.logger.Info("Stopping screenshot service")
	log.Println("Stopping screenshot service")
	if s.discordClient != nil {
		s.discordClient.Logout()
	}
	if s.uploader != nil {
		s.uploader.Close()
	}
	return nil
}

func (s *ScreenshotService) RunManually() {
	ticker := time.NewTicker(s.config.Interval)
	defer ticker.Stop()

	// Capture immediately on start if not idle
	isIdle, _ := idle.IsIdle(3 * time.Minute)
	if !isIdle {
		log.Println("Capturing first screenshot...")
		if err := s.captureAndSendWithLog(); err != nil {
			log.Printf("Error: %v\n", err)
		}
	} else {
		log.Println("User is idle, waiting for activity...")
	}

	// Then continue on interval
	for range ticker.C {
		// Check if user is idle
		idleTime, err := idle.GetIdleTime()
		if err != nil {
			log.Printf("Could not check idle time: %v\n", err)
		}

		isIdle, _ := idle.IsIdle(3 * time.Minute)
		if isIdle {
			log.Printf("User idle for %v, skipping screenshot\n", idleTime.Round(time.Second))
			continue
		}

		log.Println("Capturing screenshot...")
		if err := s.captureAndSendWithLog(); err != nil {
			log.Printf("Error: %v\n", err)
		}
	}
}

func (s *ScreenshotService) captureAndSendWithLog() error {
	screenshot, err := s.capturer.Capture()
	if err != nil {
		return err
	}

	if screenshot.ActiveWindow != nil {
		if s.logger != nil {
			s.logger.Infof("Active Window: %s (%s)", screenshot.ActiveWindow.ProcessName, screenshot.ActiveWindow.Title)
		}
		log.Printf("Active: %s (%s)\n", screenshot.ActiveWindow.ProcessName, screenshot.ActiveWindow.Title)
		if s.discordClient != nil {
			info := discord.FormatActivityPresence(screenshot.ActiveWindow.ProcessName, screenshot.ActiveWindow.Title)
			if err := s.discordClient.UpdatePresence(info); err != nil {
				if s.logger != nil {
					s.logger.Warningf("Discord Presence Error: %v", err)
				}
			}
		}
	}

	if err := s.uploader.Upload(screenshot); err != nil {
		return err
	}

	log.Println("Screenshot sent successfully")
	if s.logger != nil {
		s.logger.Info("Screenshot sent successfully")
	}
	return nil
}
