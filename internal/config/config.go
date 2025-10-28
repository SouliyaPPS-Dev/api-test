package config

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config centralises runtime configuration.
type Config struct {
	HTTPPort        string
	DatabaseURL     string
	JWTSecret       string
	JWTIssuer       string
	JWTExpiry       time.Duration
	AllowedOrigins  []string
	ReadTimeoutSec  int
	WriteTimeoutSec int
	IdleTimeoutSec  int
}

// Load reads configuration from environment variables providing sane defaults.
func Load() (Config, error) {
	if err := loadDotEnv(".env"); err != nil {
		return Config{}, fmt.Errorf("loading .env: %w", err)
	}

	cfg := Config{
		HTTPPort:        getEnv("HTTP_PORT", "8080"),
		DatabaseURL:     os.Getenv("DATABASE_URL"),
		JWTSecret:       getEnv("JWT_SECRET", ""),
		JWTIssuer:       getEnv("JWT_ISSUER", "backoffice"),
		JWTExpiry:       getDurationEnv("JWT_EXPIRY", 12*time.Hour),
		AllowedOrigins:  splitCSV(getEnv("CORS_ALLOWED_ORIGINS", "*")),
		ReadTimeoutSec:  getIntEnv("HTTP_READ_TIMEOUT", 15),
		WriteTimeoutSec: getIntEnv("HTTP_WRITE_TIMEOUT", 15),
		IdleTimeoutSec:  getIntEnv("HTTP_IDLE_TIMEOUT", 60),
	}

	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.JWTSecret == "" {
		return Config{}, fmt.Errorf("JWT_SECRET is required")
	}
	return cfg, nil
}

func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		return val
	}
	return fallback
}

func getDurationEnv(key string, fallback time.Duration) time.Duration {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			return d
		}
	}
	return fallback
}

func getIntEnv(key string, fallback int) int {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		if n, err := strconv.Atoi(val); err == nil {
			return n
		}
	}
	return fallback
}

func splitCSV(value string) []string {
	parts := []string{}
	for _, part := range strings.Split(value, ",") {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	if len(parts) == 0 {
		return []string{"*"}
	}
	return parts
}

func loadDotEnv(path string) error {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "export ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return fmt.Errorf(".env line %d: missing '='", lineNum)
		}

		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)

		if key == "" {
			return fmt.Errorf(".env line %d: empty key", lineNum)
		}

		if len(value) >= 2 {
			if (value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'') {
				value = value[1 : len(value)-1]
			}
		}

		if err := os.Setenv(key, value); err != nil {
			return fmt.Errorf(".env line %d: %w", lineNum, err)
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}
