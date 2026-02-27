package config

import (
	"os"
	"strconv"
	"time"
)

const (
	defaultPort            = "8080"
	defaultEnv             = "dev"
	defaultAppName         = "towercore-api"
	defaultLogLevel        = "info"
	defaultShutdownSeconds = 10
)

type Config struct {
	AppName         string
	Env             string
	Port            string
	LogLevel        string
	ShutdownTimeout time.Duration
}

func Load() Config {
	return Config{
		AppName:         getEnv("APP_NAME", defaultAppName),
		Env:             getEnv("APP_ENV", defaultEnv),
		Port:            getEnv("APP_PORT", defaultPort),
		LogLevel:        getEnv("LOG_LEVEL", defaultLogLevel),
		ShutdownTimeout: time.Duration(getEnvInt("SHUTDOWN_TIMEOUT_SECONDS", defaultShutdownSeconds)) * time.Second,
	}
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	v := getEnv(key, "")
	if v == "" {
		return fallback
	}

	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}
