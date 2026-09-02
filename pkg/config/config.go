package config

import (
	"os"
	"strconv"
)

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

type Storage struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	UseSSL    bool
	Region    string
}

type Postgres struct {
	User     string
	Password string
	DBName   string
}

type MinIO struct {
	RootUser     string
	RootPassword string
}

type Config struct {
	Env         string
	Port        string
	BaseURL     string
	ServiceName string
	Logger      Logger
	Database    Database
	Redis       Redis
	Storage     Storage
	Postgres    Postgres
	MinIO       MinIO
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func getEnvAsBool(key string, fallback bool) bool {
	valStr := getEnv(key, "")
	if valStr == "" {
		return fallback
	}
	val, err := strconv.ParseBool(valStr)
	if err != nil {
		return fallback
	}
	return val
}

func LoadConfig() (*Config, error) {
	baseURL := getEnv("NEXUS_API_BASE_URL", getEnv("BASE_URL", "http://localhost:3000"))

	cfg := &Config{
		Env:         getEnv("NEXUS_API_ENV", "development"),
		Port:        getEnv("NEXUS_API_PORT", "8080"),
		BaseURL:     baseURL,
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
			URL: getEnv("NEXUS_API_DB_URL", "postgres://postgres:postgres@postgres:5432/nexus?sslmode=disable"),
		},
		Redis: Redis{
			URL: getEnv("NEXUS_API_REDIS_URL", "redis://localhost:6379/0"),
		},
		Storage: Storage{
			Endpoint:  getEnv("NEXUS_API_STORAGE_ENDPOINT", "localhost:9000"),
			AccessKey: getEnv("NEXUS_API_STORAGE_ACCESS_KEY", "minioadmin"),
			SecretKey: getEnv("NEXUS_API_STORAGE_SECRET_KEY", "minioadmin"),
			UseSSL:    getEnvAsBool("NEXUS_API_STORAGE_USE_SSL", false),
			Region:    getEnv("NEXUS_API_STORAGE_REGION", "us-east-1"),
		},
		Postgres: Postgres{
			User:     getEnv("POSTGRES_USER", "postgres"),
			Password: getEnv("POSTGRES_PASSWORD", "postgres"),
			DBName:   getEnv("POSTGRES_DB", "nexus"),
		},
		MinIO: MinIO{
			RootUser:     getEnv("MINIO_ROOT_USER", "minioadmin"),
			RootPassword: getEnv("MINIO_ROOT_PASSWORD", "minioadmin"),
		},
	}

	return cfg, nil
}
