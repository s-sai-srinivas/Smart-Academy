package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	JPlagResultsDir     string
	JPlagDockerImage    string
	DBHost              string
	DBPort              string
	DBUser              string
	DBPassword          string
	DBName              string
	JWTSecret           string
	JWTIssuer           string
	JWTAudience         string
	Judge0URL           string
	SuperAdminRegdNo    string
	SuperAdminEmail     string
	SuperAdminPassword  string
	JPlagBaseDir        string
	JPlagSubmissionsDir string
	GinMode             string
	Port                string
	RedisPassword       string
	AIProvider          string
	AIAPIKey            string
	AIModel             string
	RedisPort           string
	RedisHost           string
	CORSAllowedOrigins  []string
	AIRateLimitRPM      int
	AITemperature       float64
	AIMaxTokens         int
	JPlagTimeoutSeconds int
	RedisDB             int
	JobQueueWorkers     int
	JobQueueMaxRetries  int
	JobQueueRetryDelay  int
	RedisEnabled        bool
	JobQueueEnabled     bool
}

var AppConfig *Config

func LoadConfig() error {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found: %v", err)
	}

	AppConfig = &Config{
		Port:        getEnv("PORT", "8080"),
		GinMode:     getEnv("GIN_MODE", "debug"),
		DBHost:      getEnv("DB_HOST", "localhost"),
		DBPort:      getEnv("DB_PORT", "5432"),
		DBUser:      getEnv("DB_USER", "coding_user"),
		DBPassword:  getEnv("DB_PASSWORD", ""),
		DBName:      getEnv("DB_NAME", "coding_platform"),
		JWTSecret:   getEnv("JWT_SECRET", ""),
		JWTIssuer:   getEnv("JWT_ISSUER", "coding-platform"),
		JWTAudience: getEnv("JWT_AUDIENCE", "coding-platform-api"),
		Judge0URL:   getEnv("JUDGE0_URL", "http://localhost:2358"),
		// Super admin bootstrap
		SuperAdminRegdNo:   getEnv("SUPER_ADMIN_REGDNO", ""),
		SuperAdminEmail:    getEnv("SUPER_ADMIN_EMAIL", ""),
		SuperAdminPassword: getEnv("SUPER_ADMIN_PASSWORD", ""),
		// JPlag configuration
		JPlagBaseDir:        getEnv("JPLAG_BASE_DIR", "/opt/jplag"),
		JPlagSubmissionsDir: getEnv("JPLAG_SUBMISSIONS_DIR", "/opt/jplag/submissions"),
		JPlagResultsDir:     getEnv("JPLAG_RESULTS_DIR", "/opt/jplag/results"),
		JPlagDockerImage:    getEnv("JPLAG_DOCKER_IMAGE", "jplag"),
		JPlagTimeoutSeconds: getEnvInt("JPLAG_TIMEOUT_SECONDS", 60),
		// AI configuration for quiz generation
		AIProvider:     getEnv("AI_PROVIDER", "openai"),
		AIAPIKey:       getEnv("AI_API_KEY", ""),
		AIModel:        getEnv("AI_MODEL", "gpt-4o"),
		AIMaxTokens:    getEnvInt("AI_MAX_TOKENS", 4000),
		AITemperature:  getEnvFloat("AI_TEMPERATURE", 0.7),
		AIRateLimitRPM: getEnvInt("AI_RATE_LIMIT_RPM", 10),
		// Redis/Dragonfly configuration
		RedisHost:     getEnv("REDIS_HOST", "localhost"),
		RedisPort:     getEnv("REDIS_PORT", "6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       getEnvInt("REDIS_DB", 0),
		RedisEnabled:  getEnv("REDIS_ENABLED", "false") == "true",
		// Job Queue configuration
		JobQueueEnabled:    getEnv("JOB_QUEUE_ENABLED", "false") == "true",
		JobQueueWorkers:    getEnvInt("JOB_QUEUE_WORKERS", 30),
		JobQueueMaxRetries: getEnvInt("JOB_QUEUE_MAX_RETRIES", 3),
		JobQueueRetryDelay: getEnvInt("JOB_QUEUE_RETRY_DELAY", 2),
		// CORS configuration - comma-separated list of allowed origins
		CORSAllowedOrigins: getEnvSlice("CORS_ALLOWED_ORIGINS", []string{"http://localhost:3000", "http://localhost:5173"}),
	}

	// Validate required fields
	if AppConfig.DBPassword == "" {
		return fmt.Errorf("DB_PASSWORD is required")
	}
	if AppConfig.JWTSecret == "" {
		return fmt.Errorf("JWT_SECRET is required")
	}

	return nil
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	value = strings.TrimSpace(value)
	if value == "" {
		return defaultValue
	}
	return value
}

func getEnvInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	value = strings.TrimSpace(value)
	if value == "" {
		return defaultValue
	}
	intValue, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return intValue
}

func getEnvFloat(key string, defaultValue float64) float64 {
	value := os.Getenv(key)
	value = strings.TrimSpace(value)
	if value == "" {
		return defaultValue
	}
	floatValue, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return defaultValue
	}
	return floatValue
}

func getEnvSlice(key string, defaultValue []string) []string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	// Split by comma and trim whitespace
	var result []string
	for _, v := range splitByComma(value) {
		if v != "" {
			result = append(result, v)
		}
	}
	if len(result) == 0 {
		return defaultValue
	}
	return result
}

func splitByComma(s string) []string {
	var result []string
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == ',' {
			part := trimSpace(s[start:i])
			result = append(result, part)
			start = i + 1
		}
	}
	return result
}

func trimSpace(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}

func (c *Config) GetDSN() string {
	sslmode := os.Getenv("DB_SSLMODE")
	if sslmode == "" {
		sslmode = "disable"
	}
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.DBHost, c.DBPort, c.DBUser, c.DBPassword, c.DBName, sslmode,
	)
}
