package config

import (
	"net"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Host     string
	Port     string
	DBDriver string
	DBDSN    string
}

func Load() Config {
	_ = godotenv.Load()

	return Config{
		Host:     env("HOST", "0.0.0.0"),
		Port:     env("PORT", "8080"),
		DBDriver: env("DB_DRIVER", "sqlite"),
		DBDSN:    env("DB_DSN", "./data/tsumugi.db"),
	}
}

func (c Config) Address() string {
	return net.JoinHostPort(c.Host, c.Port)
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
