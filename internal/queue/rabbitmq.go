package queue

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/go-micro/plugins/v4/broker/rabbitmq"
	"go-micro.dev/v4/broker"
)

type RabbitMQPublisher struct {
	broker     broker.Broker
	exchange   string
	routingKey string
	connected  bool
}

func NewRabbitMQPublisher(rabbitmqURL, exchange, routingKey string) (*RabbitMQPublisher, error) {
	pub := &RabbitMQPublisher{
		exchange:   exchange,
		routingKey: routingKey,
	}

	// Initialize broker
	pub.broker = rabbitmq.NewBroker(
		broker.Addrs(rabbitmqURL),
	)

	// Initialize broker
	if err := pub.broker.Init(); err != nil {
		return nil, fmt.Errorf("failed to initialize broker: %w", err)
	}

	// Connect to broker
	if err := pub.broker.Connect(); err != nil {
		return nil, fmt.Errorf("failed to connect to broker: %w", err)
	}

	pub.connected = true
	log.Printf("Connected to RabbitMQ via go-micro broker")
	log.Printf("Exchange: %s | Routing Key: %s", exchange, routingKey)

	return pub, nil
}

func (p *RabbitMQPublisher) Publish(data interface{}) error {
	if !p.connected {
		return fmt.Errorf("not connected to RabbitMQ")
	}

	// Marshal data to JSON
	body, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	log.Printf("[RABBITMQ] Publishing to Topic: %s", p.routingKey)

	// Create broker message
	message := &broker.Message{
		Header: map[string]string{
			"exchange":    p.exchange,
			"routing_key": p.routingKey,
			"timestamp":   time.Now().Format(time.RFC3339),
		},
		Body: body,
	}

	// Publish using broker (topic name is the routing key)
	if err := p.broker.Publish(p.routingKey, message); err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	// log.Printf("[RABBITMQ] Message published successfully to topic: %s", p.routingKey)
	return nil
}

func (p *RabbitMQPublisher) IsConnected() bool {
	return p.connected
}

func (p *RabbitMQPublisher) Close() error {
	if p.broker != nil {
		return p.broker.Disconnect()
	}
	return nil
}
