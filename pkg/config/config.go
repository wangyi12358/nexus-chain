package config

import (
	"fmt"

	"github.com/ilyakaznacheev/cleanenv"
)

type (
	Config struct {
		HTTP     HTTP
		DB       DB
		Chain    Chain
		RabbitMQ RabbitMQ
	}
	DB struct {
		Host     string `env:"DB_HOST"`
		Port     int    `env:"DB_PORT"`
		User     string `env:"DB_USER"`
		Password string `env:"DB_PASSWORD"`
		Name     string `env:"DB_NAME"`
		SSLMode  string `env:"DB_SSLMODE"`
	}

	HTTP struct {
		Port string `env:"HTTP_PORT"`
	}
	Chain struct {
		RpcUrl string `env:"RPC_URL"`
		WsUrl  string `env:"WS_URL"`
	}
	RabbitMQ struct {
		Enabled      bool   `env:"RABBITMQ_ENABLED" env-default:"false"`
		URL          string `env:"RABBITMQ_URL"`
		Exchange     string `env:"RABBITMQ_EXCHANGE" env-default:"nexus.events"`
		ExchangeType string `env:"RABBITMQ_EXCHANGE_TYPE" env-default:"topic"`
		Durable      bool   `env:"RABBITMQ_DURABLE" env-default:"true"`
		Queue        string `env:"RABBITMQ_QUEUE" env-default:"nexus.events.queue"`
	}
)

func New() (*Config, error) {
	cfg := Config{}
	err := cleanenv.ReadEnv(&cfg)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (c *Config) RpcUrl() (string, error) {
	if c.Chain.RpcUrl == "" {
		return "", fmt.Errorf("rpc url not configured")
	}
	return c.Chain.RpcUrl, nil
}

func (c *Config) WsUrl() (string, error) {
	if c.Chain.WsUrl == "" {
		return "", fmt.Errorf("ws url not configured")
	}
	return c.Chain.WsUrl, nil
}
