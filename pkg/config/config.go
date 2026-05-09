package config

import (
	"github.com/ilyakaznacheev/cleanenv"
)

type (
	Config struct {
		HTTP     HTTP
		DB       DB
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
