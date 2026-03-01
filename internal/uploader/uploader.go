package uploader

import (
	"fmt"
	"log"
	"time"

	"activity-tracker/internal/capture"
	"activity-tracker/internal/queue"
)

type Uploader struct {
	deviceName string
	publisher  *queue.RabbitMQPublisher
}

func New(deviceName, rabbitmqURL, exchange, routingKey string) (*Uploader, error) {
	// Initialize RabbitMQ publisher
	publisher, err := queue.NewRabbitMQPublisher(rabbitmqURL, exchange, routingKey)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize RabbitMQ: %w", err)
	}

	log.Println("RabbitMQ publisher initialized successfully")

	return &Uploader{
		deviceName: deviceName,
		publisher:  publisher,
	}, nil
}

func (u *Uploader) Upload(screenshot *capture.Screenshot) error {
	// Check connection
	if !u.publisher.IsConnected() {
		return fmt.Errorf("not connected to RabbitMQ")
	}

	// Prepare message data
	message := map[string]interface{}{
		"device_name": u.deviceName,
		"timestamp":   screenshot.Timestamp.Format(time.RFC3339),
		"image_size":  len(screenshot.ImageData),
	}

	// Add active window info
	if screenshot.ActiveWindow != nil {
		message["active_window_title"] = screenshot.ActiveWindow.Title
		message["active_process_name"] = screenshot.ActiveWindow.ProcessName
		message["active_process_id"] = screenshot.ActiveWindow.ProcessID
	}

	// TODO: Upload image to S3/storage and include URL in message
	// For now, we'll just send metadata
	// message["image_url"] = "s3://bucket/path/to/image.jpg"

	// Publish to RabbitMQ
	if err := u.publisher.Publish(message); err != nil {
		return fmt.Errorf("failed to publish to queue: %w", err)
	}

	// log.Println("Screenshot metadata published to queue")
	return nil
}

func (u *Uploader) Close() error {
	if u.publisher != nil {
		return u.publisher.Close()
	}
	return nil
}
