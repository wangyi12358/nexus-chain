package rabbitmq

import (
	"context"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

func (c *Client) publish(ctx context.Context, routingKey string, body []byte) error {
	ch, err := c.channel()
	if err != nil {
		return err
	}

	return ch.PublishWithContext(
		ctx,
		c.cfg.RabbitMQ.Exchange,
		routingKey,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Body:         body,
		},
	)
}

func (c *Client) EnsureBinding(ctx context.Context, routingKey string) error {
	if !c.cfg.RabbitMQ.Enabled || routingKey == "" {
		return nil
	}

	ch, err := c.channel()
	if err != nil {
		return err
	}

	c.mu.Lock()
	if _, ok := c.bindings[routingKey]; ok {
		c.mu.Unlock()
		return nil
	}
	c.mu.Unlock()

	if err := ch.QueueBind(
		c.cfg.RabbitMQ.Queue,
		routingKey,
		c.cfg.RabbitMQ.Exchange,
		false,
		nil,
	); err != nil {
		return fmt.Errorf("bind rabbitmq queue with key %s: %w", routingKey, err)
	}

	c.mu.Lock()
	c.bindings[routingKey] = struct{}{}
	c.mu.Unlock()
	return nil
}

func (c *Client) connect() error {
	if !c.cfg.RabbitMQ.Enabled {
		return nil
	}
	if c.cfg.RabbitMQ.URL == "" {
		return fmt.Errorf("rabbitmq url not configured")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.ch != nil && !c.ch.IsClosed() {
		return nil
	}

	conn, err := amqp.Dial(c.cfg.RabbitMQ.URL)
	if err != nil {
		return fmt.Errorf("dial rabbitmq: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("open rabbitmq channel: %w", err)
	}

	if err := ch.ExchangeDeclare(
		c.cfg.RabbitMQ.Exchange,
		c.cfg.RabbitMQ.ExchangeType,
		c.cfg.RabbitMQ.Durable,
		false,
		false,
		false,
		nil,
	); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return fmt.Errorf("declare rabbitmq exchange: %w", err)
	}

	queue, err := ch.QueueDeclare(
		c.cfg.RabbitMQ.Queue,
		c.cfg.RabbitMQ.Durable,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return fmt.Errorf("declare rabbitmq queue: %w", err)
	}

	c.conn = conn
	c.ch = ch
	c.bindings = make(map[string]struct{})
	_ = queue
	return nil
}

func (c *Client) channel() (*amqp.Channel, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.ch == nil || c.ch.IsClosed() {
		return nil, fmt.Errorf("rabbitmq channel is not ready")
	}

	return c.ch, nil
}

func (c *Client) reset() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.ch != nil {
		_ = c.ch.Close()
		c.ch = nil
	}
	if c.conn != nil {
		_ = c.conn.Close()
		c.conn = nil
	}
	c.bindings = make(map[string]struct{})
}

func (c *Client) Close() error {
	c.reset()
	return nil
}
