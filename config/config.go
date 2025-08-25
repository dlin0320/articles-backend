package config

// Config contains all configuration grouped by domain
type Config struct {
	Server     ServerConfig
	Database   DatabaseConfig
	JWT        JWTConfig
	Worker     WorkerConfig
	Logging    LoggingConfig
	Classifier ClassifierConfig
}

// All config structs use string fields only - packages handle conversion during initialization
type ServerConfig struct {
	Port         string
	Environment  string
	ReadTimeout  string
	WriteTimeout string
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

type JWTConfig struct {
	Secret     string
	Expiration string
}

type WorkerConfig struct {
	RetryInterval string
}

type LoggingConfig struct {
	Level       string
	Format      string
	ServiceName string
}

type ClassifierConfig struct {
	MinConfidenceScore string
	HTTPTimeout        string
	UserAgent          string
}
