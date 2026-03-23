package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"nexus-chain/pkg/config"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/fx"
)

type Publisher interface {
	EnsureBinding(ctx context.Context, routingKey string) error
	PublishEvent(ctx context.Context, routingKey string, payload EventMessage) error
}

type Client struct {
	cfg      *config.Config
	mu       sync.Mutex
	conn     *amqp.Connection
	ch       *amqp.Channel
	bindings map[string]struct{}
}

type EventMessage struct {
	UID             string                 `json:"uid"`
	EventID         string                 `json:"event_id"`
	ChainID         int                    `json:"chain_id"`
	ContractID      string                 `json:"contract_id"`
	ContractAddress string                 `json:"contract_address"`
	EventName       string                 `json:"event_name"`
	EventTopic      string                 `json:"event_topic"`
	RoutingKey      string                 `json:"routing_key"`
	BlockNumber     int64                  `json:"block_number"`
	TxHash          string                 `json:"tx_hash"`
	LogIndex        int64                  `json:"log_index"`
	Source          string                 `json:"source"`
	ParsedData      map[string]interface{} `json:"parsed_data"`
	PublishedAt     time.Time              `json:"published_at"`
}

func New(cfg *config.Config) *Client {
	return &Client{
		cfg:      cfg,
		bindings: make(map[string]struct{}),
	}
}

func StartServer(lc fx.Lifecycle, client *Client) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if !client.cfg.RabbitMQ.Enabled {
				return nil
			}
			return client.connect()
		},
		OnStop: func(ctx context.Context) error {
			return client.Close()
		},
	})
}

func (c *Client) PublishEvent(ctx context.Context, routingKey string, payload EventMessage) error {
	if !c.cfg.RabbitMQ.Enabled || routingKey == "" {
		return nil
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal rabbitmq payload: %w", err)
	}

	if err := c.publish(ctx, routingKey, body); err != nil {
		c.reset()
		if reconnectErr := c.connect(); reconnectErr != nil {
			return fmt.Errorf("publish rabbitmq message: %w; reconnect: %v", err, reconnectErr)
		}
		if retryErr := c.publish(ctx, routingKey, body); retryErr != nil {
			return fmt.Errorf("publish rabbitmq message after reconnect: %w", retryErr)
		}
	}

	fmt.Printf("Published RabbitMQ message to routing key %s: %s\n", routingKey, string(body))

	return nil
}
