package config

import (
	"bufio"
	"fmt"
	"net"
	neturl "net/url"
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

	httpPort := getEnv("HTTP_PORT", "")
	if httpPort == "" {
		httpPort = getEnv("PORT", "8080")
	}

	cfg := Config{
		HTTPPort:        httpPort,
		DatabaseURL:     resolveDatabaseURL(),
		JWTSecret:       getEnv("JWT_SECRET", ""),
		JWTIssuer:       getEnv("JWT_ISSUER", "backoffice"),
		JWTExpiry:       getDurationEnv("JWT_EXPIRY", 12*time.Hour),
		AllowedOrigins:  splitCSV(getEnv("CORS_ALLOWED_ORIGINS", "*")),
		ReadTimeoutSec:  getIntEnv("HTTP_READ_TIMEOUT", 15),
		WriteTimeoutSec: getIntEnv("HTTP_WRITE_TIMEOUT", 15),
		IdleTimeoutSec:  getIntEnv("HTTP_IDLE_TIMEOUT", 60),
	}

	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("database configuration missing: provide DATABASE_URL or PG* env vars")
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

func resolveDatabaseURL() string {
	for _, key := range []string{
		"DATABASE_URL",
		"DATABASE_PUBLIC_URL",
		"DATABASE_INTERNAL_URL",
		"DATABASE_EXTERNAL_URL",
		"DATABASE_URL_NO_SSL",
		"DATABASE_DIRECT_URL",
		"POSTGRES_URL",
		"PGURL",
		"RAILWAY_DATABASE_URL",
		"RAILWAY_PUBLIC_URL",
	} {
		if url := os.Getenv(key); url != "" {
			if coerced := coerceDatabaseURL(url); coerced != "" {
				return coerced
			}
		}
	}

	for _, key := range []string{"DATABASE_URL_FILE", "PGURL_FILE"} {
		if urlFromFile := readEnvFile(key); urlFromFile != "" {
			if coerced := coerceDatabaseURL(urlFromFile); coerced != "" {
				return coerced
			}
		}
	}

	host := firstNonEmpty(
		os.Getenv("PGHOST"),
		os.Getenv("POSTGRES_HOST"),
		os.Getenv("POSTGRESQL_ADDON_HOST"),
		os.Getenv("DATABASE_HOST"),
		os.Getenv("RAILWAY_TCP_PROXY_DOMAIN"),
		os.Getenv("RAILWAY_PRIVATE_DOMAIN"),
	)
	user := firstNonEmpty(
		os.Getenv("PGUSER"),
		os.Getenv("POSTGRES_USER"),
		os.Getenv("POSTGRESQL_ADDON_USER"),
		os.Getenv("DATABASE_USERNAME"),
		os.Getenv("DATABASE_USER"),
	)
	if user == "" {
		if dbUser := os.Getenv("DATABASE_USER"); dbUser != "" {
			user = dbUser
		}
	}
	password := firstNonEmpty(
		os.Getenv("PGPASSWORD"),
		os.Getenv("POSTGRES_PASSWORD"),
		os.Getenv("POSTGRESQL_ADDON_PASSWORD"),
		os.Getenv("DATABASE_PASSWORD"),
	)
	if password == "" {
		password = os.Getenv("DATABASE_PASSWORD")
	}
	database := firstNonEmpty(
		os.Getenv("PGDATABASE"),
		os.Getenv("POSTGRES_DB"),
		os.Getenv("POSTGRES_DATABASE"),
		os.Getenv("POSTGRESQL_ADDON_DB"),
		os.Getenv("DATABASE_NAME"),
	)
	port := firstNonEmpty(
		os.Getenv("PGPORT"),
		os.Getenv("POSTGRES_PORT"),
		os.Getenv("POSTGRESQL_ADDON_PORT"),
		os.Getenv("DATABASE_PORT"),
		os.Getenv("RAILWAY_TCP_PROXY_PORT"),
	)
	if port == "" {
		port = "5432"
	}
	sslMode := firstNonEmpty(
		os.Getenv("PGSSLMODE"),
		os.Getenv("PGSSL_MODE"),
		os.Getenv("PGSSL"),
		os.Getenv("POSTGRES_SSL_MODE"),
		"require",
	)

	if database == "" {
		database = firstNonEmpty(user, "postgres")
	}

	dsn := &neturl.URL{
		Scheme: "postgres",
		Path:   "/" + database,
	}

	// Allow host to be empty only if we previously returned.
	if host == "" {
		return ""
	}
	dsn.Host = net.JoinHostPort(host, port)

	if user == "" {
		return ""
	}
	dsn.User = neturl.User(user)
	if password != "" {
		dsn.User = neturl.UserPassword(user, password)
	}

	query := dsn.Query()
	if sslMode != "" && query.Get("sslmode") == "" {
		query.Set("sslmode", sslMode)
	}
	dsn.RawQuery = query.Encode()

	return normalisePostgresScheme(dsn.String())
}

func normalisePostgresScheme(url string) string {
	if strings.HasPrefix(url, "postgresql://") {
		return "postgres://" + strings.TrimPrefix(url, "postgresql://")
	}
	return url
}

func coerceDatabaseURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(raw, "postgres://") || strings.HasPrefix(raw, "postgresql://") {
		return normalisePostgresScheme(raw)
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func readEnvFile(key string) string {
	path := os.Getenv(key)
	if path == "" {
		return ""
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
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
