package config

// DatabaseConfig contém os parâmetros de ligação à base de dados do AIP.
// Nota: o AIP tem a sua própria base (ai_features, ai_predictions,
// ai_events, ai_models) — não é a mesma base do towercore.
type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
}

func LoadDatabase() DatabaseConfig {
	return DatabaseConfig{
		Host:     getEnv("AIP_DB_HOST", "localhost"),
		Port:     getEnv("AIP_DB_PORT", "5432"),
		User:     getEnv("AIP_DB_USER", "postgres"),
		Password: getEnv("AIP_DB_PASSWORD", "postgres"),
		Name:     getEnv("AIP_DB_NAME", "aip"),
		SSLMode:  getEnv("AIP_DB_SSLMODE", "disable"),
	}
}

// DSN devolve a connection string no formato esperado por lib/pq.
func (c DatabaseConfig) DSN() string {
	return "host=" + c.Host +
		" port=" + c.Port +
		" user=" + c.User +
		" password=" + c.Password +
		" dbname=" + c.Name +
		" sslmode=" + c.SSLMode +
		" client_encoding=UTF8"
}
