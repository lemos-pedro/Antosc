package config

import "os"

// Config agrega toda a configuração do AIP (Antosc Intelligence Platform).
type Config struct {
	AppEnv       string
	LogLevel     string
	APIPort      string
	TowerCoreURL string

	// PredictionServiceURL aponta para o microserviço Python responsável por
	// treino/inferência (features_engineering -> train -> inference).
	// O Go nunca corre modelos ML diretamente; apenas consome previsões
	// já calculadas via este contrato HTTP.
	PredictionServiceURL string

	Database DatabaseConfig
}

// Load lê a configuração a partir de variáveis de ambiente, com defaults
// seguros para desenvolvimento local.
func Load() Config {
	return Config{
		AppEnv:               getEnv("AIP_ENV", "dev"),
		LogLevel:             getEnv("AIP_LOG_LEVEL", "info"),
		APIPort:              getEnv("AIP_API_PORT", "8090"),
		TowerCoreURL:         getEnv("TOWERCORE_URL", "http://localhost:8080"),
		PredictionServiceURL: getEnv("AIP_PREDICTION_SERVICE_URL", "http://localhost:9000"),
		Database:             LoadDatabase(),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
