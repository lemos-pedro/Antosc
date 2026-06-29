package config

import (
	"os"
	"strconv"
	"strings"
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
	Cache           CacheConfig
	DB              DBConfig
	Scheduler       SchedulerConfig
	SNMP            SNMPConfig
	Discovery       DiscoveryConfig
	Auth            AuthConfig
	RateLimit       RateLimitConfig
}

type CacheConfig struct {
	TowerListTTLSeconds   int
	TowerDetailTTLSeconds int
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
	Enabled         bool
	IntervalSeconds int
	BatchSize       int
}

type SNMPConfig struct {
	TimeoutSeconds    int
	Retries           int
	SecretKey         string
	TowersJSON        string
	SyntheticFallback bool
}

// DiscoveryConfig controla o network discovery SNMP: varrimento
// periódico de um CIDR à procura de agentes a responder, registados
// em discovered_devices para revisão/promoção manual.
type DiscoveryConfig struct {
	Enabled         bool
	CIDR            string
	Community       string
	IntervalSeconds int
	Concurrency     int
	TimeoutMillis   int
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
		Cache: CacheConfig{
			TowerListTTLSeconds:   getEnvIntAllowZero("CACHE_TOWER_LIST_TTL_SECONDS", 5),
			TowerDetailTTLSeconds: getEnvIntAllowZero("CACHE_TOWER_DETAIL_TTL_SECONDS", 10),
		},
		DB: DBConfig{
			Host:                   getEnv("DB_HOST", "localhost"),
			Port:                   getEnv("DB_PORT", "5432"),
			Name:                   getEnv("DB_NAME", "towercore"),
			User:                   getEnv("DB_USER", "A.lemos"),
			Password:               getEnv("DB_PASSWORD", "A.lemos13:7002"),
			SSLMode:                getEnv("DB_SSL_MODE", "disable"),
			MaxOpenConns:           getEnvInt("DB_MAX_OPEN_CONNS", 10),
			MaxIdleConns:           getEnvInt("DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetimeMinutes: getEnvInt("DB_CONN_MAX_LIFETIME_MINUTES", 30),
		},
		Scheduler: SchedulerConfig{
			Enabled:         getEnvBool("SCHEDULER_ENABLED", true),
			IntervalSeconds: getEnvInt("POLL_INTERVAL_SECONDS", 60),
			BatchSize:       getEnvInt("POLL_BATCH_SIZE", 100),
		},
		SNMP: SNMPConfig{
			TimeoutSeconds:    getEnvInt("SNMP_TIMEOUT_SECONDS", 10),
			Retries:           getEnvInt("SNMP_RETRIES", 1),
			SecretKey:         getEnv("SNMP_SECRET_KEY", ""),
			TowersJSON:        getEnv("SNMP_TOWERS_JSON", ""),
			SyntheticFallback: getEnvBool("SNMP_SYNTHETIC_FALLBACK", false),
		},
		Discovery: DiscoveryConfig{
			Enabled:         getEnvBool("DISCOVERY_ENABLED", false),
			CIDR:            getEnv("DISCOVERY_CIDR", "10.0.0.0/24"),
			Community:       getEnv("DISCOVERY_COMMUNITY", "Antosc-noc"),
			IntervalSeconds: getEnvInt("DISCOVERY_INTERVAL_SECONDS", 600),
			Concurrency:     getEnvInt("DISCOVERY_CONCURRENCY", 32),
			TimeoutMillis:   getEnvInt("DISCOVERY_TIMEOUT_MILLIS", 800),
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

func getEnvBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}

	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}
