package config

import "os"

// Load reads configuration from environment variables as raw strings
// Components handle validation and defaults during initialization
func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Port:         os.Getenv("SERVER_PORT"),
			Environment:  os.Getenv("SERVER_ENV"),
			ReadTimeout:  os.Getenv("SERVER_READ_TIMEOUT"),
			WriteTimeout: os.Getenv("SERVER_WRITE_TIMEOUT"),
		},
		Database: DatabaseConfig{
			Host:     os.Getenv("DB_HOST"),
			Port:     os.Getenv("DB_PORT"),
			User:     os.Getenv("DB_USER"),
			Password: os.Getenv("DB_PASSWORD"),
			DBName:   os.Getenv("DB_NAME"),
			SSLMode:  os.Getenv("DB_SSLMODE"),
		},
		JWT: JWTConfig{
			Secret:     os.Getenv("JWT_SECRET"),
			Expiration: os.Getenv("JWT_EXPIRATION"),
		},
		Worker: WorkerConfig{
			RetryInterval: os.Getenv("WORKER_RETRY_INTERVAL"),
		},
		Logging: LoggingConfig{
			Level:       os.Getenv("LOG_LEVEL"),
			Format:      os.Getenv("LOG_FORMAT"),
			ServiceName: os.Getenv("SERVICE_NAME"),
		},
		Classifier: ClassifierConfig{
			MinConfidenceScore: os.Getenv("CLASSIFIER_MIN_CONFIDENCE"),
			HTTPTimeout:        os.Getenv("CLASSIFIER_HTTP_TIMEOUT"),
			UserAgent:          os.Getenv("CLASSIFIER_USER_AGENT"),
		},
	}
}
