package config

import "os"

// Config agrega toda a configuração do AIP (Antosc Intelligence Platform).
type Config struct {
	AppEnv       string
	LogLevel     string
	APIPort      string
	TowerCoreURL string

	// GeneratorAPIURL é o serviço ComAp de energia do gerador (combustível,
	// horas de funcionamento) -- serviço separado do towercore, porta própria.
	GeneratorAPIURL string

	// PredictionServiceURL aponta para o microserviço Python responsável por
	// treino/inferência (features_engineering -> train -> inference).
	// O Go nunca corre modelos ML diretamente; apenas consome previsões
	// já calculadas via este contrato HTTP.
	PredictionServiceURL string

	// AIPBaseURL é o endereço da própria API do AIP (este processo), usado
	// pelo assistente para chamar as suas ferramentas via HTTP em vez de
	// aceder à base de dados diretamente. Mantém uma única fronteira de
	// acesso a dados (a API), com o Ollama sempre do lado de fora dela.
	AIPBaseURL string

	// OllamaURL e OllamaModel configuram o modelo local usado pelo
	// assistente. Correr localmente evita custo por pedido/token.
	OllamaURL   string
	OllamaModel string

	// Resend envia o relatório semanal por email (plano gratuito até
	// 3000 emails/mês). Deixa ResendAPIKey vazio para desligar o envio.
	ResendAPIKey string
	ResendFrom   string

	// TeamsWebhookURL recebe alertas de limiar em tempo real (bateria,
	// combustível, etc.). Deixa vazio para desligar o envio ao Teams.
	TeamsWebhookURL string

	// JWTSecret assina tokens de utilizadores. Mínimo 32 chars em produção.
	JWTSecret string
	// JWTTTLHours validade do access token (default 1 hora).
	JWTTTLHours int
	// RefreshTTLHours validade do refresh token (default 168 = 7 dias).
	RefreshTTLHours int
	// MaxFailedLogins antes de lockout (default 5).
	MaxFailedLogins int
	// LockoutMinutes duração do bloqueio (default 15).
	LockoutMinutes int

	// APIKey protege Power BI/exports (alternativa a JWT).
	APIKey string

	Database DatabaseConfig
}

// Load lê a configuração a partir de variáveis de ambiente, com defaults
// seguros para desenvolvimento local.
func Load() Config {
	return Config{
		AppEnv:               getEnv("AIP_ENV", "dev"),
		LogLevel:             getEnv("AIP_LOG_LEVEL", "info"),
		APIPort:              getEnv("AIP_API_PORT", "8090"),
		TowerCoreURL:         getEnv("TOWERCORE_URL", "http://localhost:8000"),
		GeneratorAPIURL:      getEnv("GENERATOR_API_URL", "http://localhost:8001"),
		PredictionServiceURL: getEnv("AIP_PREDICTION_SERVICE_URL", "http://localhost:8001"),
		AIPBaseURL:           getEnv("AIP_BASE_URL", "http://localhost:8090"),
		OllamaURL:            getEnv("OLLAMA_URL", "http://localhost:11434"),
		OllamaModel:          getEnv("OLLAMA_MODEL", "qwen2.5:3b"),
		ResendAPIKey:         getEnv("RESEND_API_KEY", ""),
		ResendFrom:           getEnv("RESEND_FROM", "alerts@antosc.com"),
		TeamsWebhookURL:      getEnv("TEAMS_WEBHOOK_URL", ""),
		JWTSecret:            getEnv("AIP_JWT_SECRET", ""),
		JWTTTLHours:          getEnvInt("AIP_JWT_TTL_HOURS", 1),
		RefreshTTLHours:      getEnvInt("AIP_REFRESH_TTL_HOURS", 168),
		MaxFailedLogins:      getEnvInt("AIP_MAX_FAILED_LOGINS", 5),
		LockoutMinutes:       getEnvInt("AIP_LOCKOUT_MINUTES", 15),
		APIKey:               getEnv("AIP_API_KEY", ""),
		Database:             LoadDatabase(),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n := 0
	for _, c := range v {
		if c < '0' || c > '9' {
			return fallback
		}
		n = n*10 + int(c-'0')
	}
	if n <= 0 {
		return fallback
	}
	return n
}
