package config

import (
	"flag"
	"os"
	"strconv"
)

type Config struct {
	Host         string
	Port         int
	DatabasePath string
	LogLevel     string
}

func Load() *Config {
	cfg := &Config{
		Host:         "127.0.0.1",
		Port:         8088,
		DatabasePath: "llmrouter.db",
		LogLevel:     "INFO",
	}

	if envHost := os.Getenv("LLMROUTER_HOST"); envHost != "" {
		cfg.Host = envHost
	}

	if envPort := os.Getenv("LLMROUTER_PORT"); envPort != "" {
		if p, err := strconv.Atoi(envPort); err == nil {
			cfg.Port = p
		}
	}

	if envDB := os.Getenv("LLMROUTER_DB_PATH"); envDB != "" {
		cfg.DatabasePath = envDB
	}

	if envLog := os.Getenv("LLMROUTER_LOG_LEVEL"); envLog != "" {
		cfg.LogLevel = envLog
	}

	// CLI flags override env if provided
	if flag.Lookup("port") == nil {
		flag.StringVar(&cfg.Host, "host", cfg.Host, "Host IP to bind HTTP server")
		flag.IntVar(&cfg.Port, "port", cfg.Port, "Port to bind HTTP server")
		flag.StringVar(&cfg.DatabasePath, "db", cfg.DatabasePath, "Path to SQLite database file")
		flag.StringVar(&cfg.LogLevel, "log-level", cfg.LogLevel, "Log level (DEBUG, INFO, WARN, ERROR)")
	}

	return cfg
}
