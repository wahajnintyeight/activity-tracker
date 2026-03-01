package main

import (
	"log"
	"os"

	"activity-tracker/internal/config"
	"activity-tracker/internal/service"

	svc "github.com/kardianos/service"
)

func main() {
	// If no arguments, run in background mode
	if len(os.Args) == 1 {
		cfg := config.Load()
		deviceName, _ := os.Hostname()
		fullService, err := service.New(cfg, deviceName)
		if err != nil {
			log.Fatal(err)
		}
		fullService.RunManually()
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
		log.Fatalf("Unknown command: %s\nAvailable: install, uninstall, start, stop, restart, run", cmd)
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
			log.Printf("Failed to initialize service: %v", err)
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
