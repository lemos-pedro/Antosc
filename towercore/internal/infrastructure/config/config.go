package config

import (
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

const (
	defaultPort            = "8000"
	defaultEnv             = "dev"
	defaultAppName         = "towercore-api"
	defaultLogLevel        = "info"
	defaultShutdownSeconds = 10
	defaultEnvFile         = ".env"
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
	Nagios          NagiosConfig
	Comap           ComapConfig
	NetEco          NetEco
	Zabbix          ZabbixConfig
	Hizima		    HizimaConfig
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

type NagiosConfig struct {
	BaseURL             string
	Username            string
	Password            string
	TimeoutSeconds      int
	PollIntervalSeconds int
}

// NetEco controla a integração com o Huawei iManager NetEco: polling de
// bateria/energia via API interna (login por sessão) e receção de alarmes
// via SNMP trap. Enabled=false por default — depende de credenciais de
// sessão válidas, não é NBI oficial licenciado.
type NetEco struct {
	Enabled               bool
	BaseURL               string
	Username              string
	Password              string
	IntervalSeconds       int
	TimeoutSeconds        int
	TLSInsecureSkipVerify bool
	TrapPort              int
	TrapCommunity         string
}

// ComapConfig controla o polling Modbus dos controladores de grupo gerador
// ComAp. Registos ainda não validados em campo (fuel_percent,
// battery_voltage — ver adapters/comap/profile.go) continuam a ser lidos e
// persistidos, mas não geram eventos/tickets enquanto StatusUnconfirmed.
type ComapConfig struct {
	Enabled         bool
	IntervalSeconds int
	TimeoutSeconds  int
	Retries         int
}

// ZabbixConfig controla a integração com a API JSON-RPC do Zabbix, usada
// para sincronizar network_links + link_metric_snapshots (ver
// adapters/zabbix e services.ZabbixLinkSyncService). TimeoutSeconds
// alimenta o http.Client do zabbix.Client (chamadas host.get/item.get);
// IntervalSeconds controla a cadência do scheduler.ZabbixScheduler.
type ZabbixConfig struct {
	Enabled         bool
	BaseURL         string
	Username        string
	Password        string
	APIToken        string
	HostSearch      string
	TimeoutSeconds  int
	IntervalSeconds int
}


type HizimaConfig struct {
	Enabled    bool   // HIZIMA_ENABLED — reservado para uso futuro (scheduler de alarmes)
	Host       string // HIZIMA_HOST, ex: "https://antosc.hizima.com"
	ClientID   string // HIZIMA_CLIENT_ID, ex: "anglobal"
	Security   string // HIZIMA_SECURITY — segredo, nunca versionar
	Username   string // HIZIMA_USERNAME
	Password   string // HIZIMA_PASSWORD
	StationMap string 
}

func Load() Config {
	loadEnvFile()

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
		Nagios: NagiosConfig{
			BaseURL:             getEnv("NAGIOS_BASE_URL", "http://172.17.0.31"),
			Username:            getEnv("NAGIOS_USER", ""),
			Password:            getEnv("NAGIOS_PASSWORD", ""),
			TimeoutSeconds:      getEnvInt("NAGIOS_TIMEOUT_SECONDS", 10),
			PollIntervalSeconds: getEnvInt("NAGIOS_POLL_INTERVAL_SECONDS", 60),
		},
		Comap: ComapConfig{
			Enabled:         getEnvBool("COMAP_ENABLED", true),
			IntervalSeconds: getEnvInt("COMAP_POLL_INTERVAL_SECONDS", 60),
			TimeoutSeconds:  getEnvInt("MODBUS_TIMEOUT_SECONDS", 2),
			Retries:         getEnvInt("MODBUS_RETRIES", 1),
		},
		NetEco: NetEco{
			Enabled:               getEnvBool("NETECO_ENABLED", false),
			BaseURL:               getEnv("NETECO_BASE_URL", "https://192.168.9.11:31943"),
			Username:              getEnv("NETECO_USERNAME", ""),
			Password:              getEnv("NETECO_PASSWORD", ""),
			IntervalSeconds:       getEnvInt("NETECO_POLL_INTERVAL_SECONDS", 60),
			TimeoutSeconds:        getEnvInt("NETECO_TIMEOUT_SECONDS", 10),
			TLSInsecureSkipVerify: getEnvBool("NETECO_TLS_INSECURE_SKIP_VERIFY", true),
			TrapPort:              getEnvInt("NETECO_TRAP_PORT", 162),
			TrapCommunity:         getEnv("NETECO_TRAP_COMMUNITY", "TowercoreRead1"),
		},
		Zabbix: ZabbixConfig{
			Enabled:         getEnvBool("ZABBIX_ENABLED", false),
			BaseURL:         getEnv("ZABBIX_URL", "http://192.168.116.10/zabbix"),
			Username:        getEnv("ZABBIX_USERNAME", ""),
			Password:        getEnv("ZABBIX_PASSWORD", ""),
			APIToken:        getEnv("ZABBIX_API_TOKEN", ""),
			HostSearch:      getEnv("ZABBIX_HOST_SEARCH", ""),
			TimeoutSeconds:  getEnvInt("ZABBIX_TIMEOUT_SECONDS", 10),
			IntervalSeconds: getEnvInt("ZABBIX_POLL_INTERVAL_SECONDS", 60),
		},
		Hizima: HizimaConfig{
			Enabled:    getEnvBool("HIZIMA_ENABLED", false),
			Host:       getEnv("HIZIMA_HOST", ""),
			ClientID:   getEnv("HIZIMA_CLIENT_ID", ""),
			Security:   getEnv("HIZIMA_SECURITY", ""),
			Username:   getEnv("HIZIMA_USERNAME", ""),
			Password:   getEnv("HIZIMA_PASSWORD", ""),
			StationMap: getEnv("HIZIMA_STATION_MAP", ""),
		},
	}
}

// loadEnvFile tenta carregar variáveis de um ficheiro .env para o
// ambiente do processo, ANTES de qualquer getEnv ser chamado.
//
// Comportamento:
//   - Não sobrepõe variáveis já definidas no ambiente do processo
//     (ex.: via export/$env:, ou injetadas a sério em produção/CI).
//   - Caminho configurável via ENV_FILE (útil se o .env não estiver
//     no diretório de onde corres `go run`/o binário).
//   - Se o ficheiro não existir, não é erro fatal — assume-se que o
//     ambiente já tem as variáveis necessárias (caso de produção).
//   - Regista sempre no log o que aconteceu, para nunca mais isto
//     falhar em silêncio.
func loadEnvFile() {
	path := os.Getenv("ENV_FILE")
	if path == "" {
		path = defaultEnvFile
	}

	if err := godotenv.Load(path); err != nil {
		if os.IsNotExist(err) {
			log.Printf("CONFIG: %s não encontrado (cwd atual) — a usar apenas variáveis já presentes no ambiente", path)
			return
		}
		log.Printf("CONFIG: falha ao ler %s: %v", path, err)
		return
	}

	log.Printf("CONFIG: variáveis carregadas de %s", path)
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