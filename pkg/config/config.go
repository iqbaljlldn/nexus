package config

import "os"

type Logger struct {
	Level      string
	LogFile    string
	MaxSize    int
	MaxBackups int
	MaxAge     int
	Compress   bool
}

type Database struct {
	URL string
}

type Redis struct {
	URL string
}

type Config struct {
	Env         string
	Port        string
	Logger      Logger
	ServiceName string
	Database    Database
	Redis       Redis
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func LoadConfig() (*Config, error) {
	cfg := &Config{
		Env:         getEnv("NEXUS_API_ENV", "development"),
		Port:        getEnv("NEXUS_API_PORT", "8080"),
		ServiceName: getEnv("NEXUS_API_SERVICE_NAME", "nexus"),
		Logger: Logger{
			Level:      getEnv("NEXUS_API_LOG_LEVEL", "debug"),
			LogFile:    "logs/nexus.log",
			MaxSize:    10,
			MaxBackups: 3,
			MaxAge:     28,
			Compress:   true,
		},
		Database: Database{
			URL: getEnv("NEXUS_API_DB_URL", "postgres://nexus:nexuspassword@localhost:5432/nexus?sslmode=disable"),
		},
		Redis: Redis{
			URL: getEnv("NEXUS_API_REDIS_URL", "redis://localhost:6379/0"),
		},
	}

	return cfg, nil
}
