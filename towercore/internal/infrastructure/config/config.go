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
	DB              DBConfig
	Scheduler       SchedulerConfig
	SNMP            SNMPConfig
	Auth            AuthConfig
	RateLimit       RateLimitConfig
}

type DBConfig struct {
	Host                   string
	Port                   string
	Name                   string
	User                   string
	Password               string
	SSLMode                string
	MaxOpenConns           int
	MaxIdleConns           int
	ConnMaxLifetimeMinutes int
}

type SchedulerConfig struct {
	IntervalSeconds int
	BatchSize       int
}

type SNMPConfig struct {
	TimeoutSeconds int
	Retries        int
	SecretKey      string
}

type AuthConfig struct {
	APIKeyHash        string
	BearerToken       string
	UserTokenSecret   string
	UserTokenTTLMin   int
	BootstrapUsername string
	BootstrapPassword string
	BootstrapRole     string
}

type RateLimitConfig struct {
	WritePerMinute int
}

func Load() Config {
	return Config{
		AppName:         getEnv("APP_NAME", defaultAppName),
		Env:             getEnv("APP_ENV", defaultEnv),
		Port:            getEnv("APP_PORT", defaultPort),
		LogLevel:        getEnv("LOG_LEVEL", defaultLogLevel),
		ShutdownTimeout: time.Duration(getEnvInt("SHUTDOWN_TIMEOUT_SECONDS", defaultShutdownSeconds)) * time.Second,
		DB: DBConfig{
			Host:                   getEnv("DB_HOST", "localhost"),
			Port:                   getEnv("DB_PORT", "5432"),
			Name:                   getEnv("DB_NAME", "antosc"),
			User:                   getEnv("DB_USER", "postgres"),
			Password:               getEnv("DB_PASSWORD", "postgres"),
			SSLMode:                getEnv("DB_SSL_MODE", "disable"),
			MaxOpenConns:           getEnvInt("DB_MAX_OPEN_CONNS", 10),
			MaxIdleConns:           getEnvInt("DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetimeMinutes: getEnvInt("DB_CONN_MAX_LIFETIME_MINUTES", 30),
		},
		Scheduler: SchedulerConfig{
			IntervalSeconds: getEnvInt("POLL_INTERVAL_SECONDS", 60),
			BatchSize:       getEnvInt("POLL_BATCH_SIZE", 100),
		},
		SNMP: SNMPConfig{
			TimeoutSeconds: getEnvInt("SNMP_TIMEOUT_SECONDS", 10),
			Retries:        getEnvInt("SNMP_RETRIES", 1),
			SecretKey:      getEnv("SNMP_SECRET_KEY", ""),
		},
		Auth: AuthConfig{
			APIKeyHash:        getEnv("AUTH_API_KEY_HASH", ""),
			BearerToken:       getEnv("AUTH_BEARER_TOKEN", ""),
			UserTokenSecret:   getEnv("AUTH_USER_TOKEN_SECRET", ""),
			UserTokenTTLMin:   getEnvInt("AUTH_USER_TOKEN_TTL_MIN", 480),
			BootstrapUsername: getEnv("AUTH_BOOTSTRAP_USERNAME", ""),
			BootstrapPassword: getEnv("AUTH_BOOTSTRAP_PASSWORD", ""),
			BootstrapRole:     getEnv("AUTH_BOOTSTRAP_ROLE", "admin"),
		},
		RateLimit: RateLimitConfig{
			WritePerMinute: getEnvIntAllowZero("RATE_LIMIT_WRITE_PER_MINUTE", 120),
		},
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

func getEnvIntAllowZero(key string, fallback int) int {
	v := getEnv(key, "")
	if v == "" {
		return fallback
	}

	n, err := strconv.Atoi(v)
	if err != nil || n < 0 {
		return fallback
	}
	return n
}
