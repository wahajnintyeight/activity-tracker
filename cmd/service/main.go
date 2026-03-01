package main

import (
	"log"
	"os"

	"activity-tracker/internal/config"
	"activity-tracker/internal/service"

	svc "github.com/kardianos/service"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Get device name
	deviceName, err := os.Hostname()
	if err != nil {
		log.Fatal(err)
	}

	// Service configuration
	svcConfig := &svc.Config{
		Name:        "ScreenshotService",
		DisplayName: "Screenshot Capture Service",
		Description: "Captures and uploads screenshots periodically",
	}

	// Create service instance
	prg, err := service.New(cfg, deviceName)
	if err != nil {
		log.Fatalf("Failed to create service: %v", err)
	}

	s, err := svc.New(prg, svcConfig)
	if err != nil {
		log.Fatal(err)
	}

	logger, err := s.Logger(nil)
	if err != nil {
		log.Fatal(err)
	}
	prg.SetLogger(logger)

	// Handle commands
	if len(os.Args) > 1 {
		handleCommand(os.Args[1], s, prg, cfg, deviceName)
		return
	}

	// Run as service (when started by Windows Service Manager)
	if err := s.Run(); err != nil {
		log.Fatal(err)
	}
}

func handleCommand(cmd string, s svc.Service, prg *service.ScreenshotService, cfg *config.Config, deviceName string) {
	var err error

	switch cmd {
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
	case "run":
		log.Printf("Running manually on device: %s\n", deviceName)
		log.Printf("Capturing every %v, publishing to RabbitMQ exchange: %s\n", cfg.Interval, cfg.RabbitMQExchange)
		log.Println("Press Ctrl+C to stop")
		prg.RunManually()
	default:
		log.Fatalf("Unknown command: %s\nAvailable: install, uninstall, start, stop, restart, run", cmd)
	}

	if err != nil {
		log.Fatal(err)
	}
}
