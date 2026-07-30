package config

type Logger struct {
	Level      string
	LogFile    string
	MaxSize    int
	MaxBackups int
	MaxAge     int
	Compress   bool
}

type Config struct {
	Env         string
	Port        string
	Logger      Logger
	ServiceName string
}

func LoadConfig() (*Config, error) {
	cfg := &Config{
		Env:         "development",
		Port:        "8080",
		ServiceName: "nexus",
		Logger: Logger{
			Level:      "debug",
			LogFile:    "logs/nexus.log",
			MaxSize:    10,
			MaxBackups: 3,
			MaxAge:     28,
			Compress:   true,
		},
	}

	return cfg, nil
}
