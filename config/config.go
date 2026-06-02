// Package config loads application settings from environment variables.
// In development, values are read from a .env file via godotenv.
// In production, set the variables directly in your environment or secret manager.
package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config holds every setting the application needs to start.
type Config struct {
	// HTTP
	Port string

	// MySQL
	DBHost     string
	DBPort     string
	DBName     string
	DBUser     string
	DBPassword string

	// JWT
	JWTSecret          string
	JWTAccessTokenTTL  int // minutes
	JWTRefreshTokenTTL int // days

	// Shiftbase
	ShiftbaseAPIKey              string
	ShiftbaseBaseURL             string
	ShiftbaseCreateEmployeePath  string
	ShiftbaseDefaultDepartmentID int

	// Server
	ServerBaseURL string

	// S3-compatible object storage (Hetzner)
	S3Endpoint  string
	S3AccessKey string
	S3SecretKey string
	S3Bucket    string
	S3Region    string
}

// Load reads .env (if present) and then reads all required variables from the environment.
// It returns an error listing every missing required variable so you can fix them all at once.
func Load() (*Config, error) {
	// .env is optional — in production the variables will already be set
	_ = godotenv.Load()

	var missing []string

	require := func(key string) string {
		v := os.Getenv(key)
		if v == "" {
			missing = append(missing, key)
		}
		return v
	}

	cfg := &Config{
		Port:                        getEnv("PORT", "8080"),
		DBHost:                      getEnv("DB_HOST", "127.0.0.1"),
		DBPort:                      getEnv("DB_PORT", "3306"),
		DBName:                      require("DB_NAME"),
		DBUser:                      require("DB_USER"),
		DBPassword:                  require("DB_PASSWORD"),
		JWTSecret:                   require("JWT_SECRET"),
		ShiftbaseAPIKey:              getEnv("SHIFTBASE_API_KEY", ""),
		ShiftbaseBaseURL:             getEnv("SHIFTBASE_BASE_URL", "https://api.shiftbase.com"),
		ShiftbaseCreateEmployeePath:  getEnv("SHIFTBASE_CREATE_EMPLOYEE_PATH", "/users"),
		ShiftbaseDefaultDepartmentID: envInt("SHIFTBASE_DEFAULT_DEPARTMENT_ID", 0),
		ServerBaseURL:               getEnv("SERVER_BASE_URL", "http://localhost:8080"),
		S3Endpoint:                  getEnv("S3_ENDPOINT", ""),
		S3AccessKey:                 getEnv("S3_ACCESS_KEY", ""),
		S3SecretKey:                 getEnv("S3_SECRET_KEY", ""),
		S3Bucket:                    getEnv("S3_BUCKET", ""),
		S3Region:                    getEnv("S3_REGION", "eu-central"),
		JWTAccessTokenTTL:           envInt("JWT_ACCESS_TTL_MINUTES", 15),
		JWTRefreshTokenTTL:          envInt("JWT_REFRESH_TTL_DAYS", 30),
	}

	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required environment variables: %v", missing)
	}

	return cfg, nil
}

// DSN returns a MySQL data source name ready for sql.Open.
func (c *Config) DSN() string {
	return fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?parseTime=true&loc=Europe%%2FAmsterdam&charset=utf8mb4",
		c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName,
	)
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}
