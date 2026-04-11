package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Port                        string
	CORSAllowedOrigins          []string
	AdminAPIKeys                []string
	DatabaseURL                 string
	DBMaxOpenConns              int
	DBMaxIdleConns              int
	DBConnMaxLifetime           time.Duration
	DBConnMaxIdleTime           time.Duration
	AliExpressAppKey            string
	AliExpressAppSecret         string
	AliExpressDSAppKey          string
	AliExpressDSAppSecret       string
	PriceUpdateScheduleEnabled  bool
	PriceUpdateScheduleInterval time.Duration
	SKUEnrichMinDelay           time.Duration
	SKUEnrichMaxDelay           time.Duration
	SKUSnapshotMinDelay         time.Duration
	SKUSnapshotMaxDelay         time.Duration
	TokenRefreshEnabled         bool
	TokenRefreshInterval        time.Duration
	HotProductScheduleEnabled   bool
	HotProductScheduleInterval  time.Duration
	HotProductSnapshotStagger   time.Duration
	SessionCleanupEnabled       bool
	SessionCleanupInterval      time.Duration
	SessionCleanupRetentionDays int
}

func Load() Config {
	return Config{
		Port:                        getEnvOrDefault("PORT", "8700"),
		CORSAllowedOrigins:          getEnvAsCSV("CORS_ALLOWED_ORIGINS", []string{"http://localhost:5173", "http://127.0.0.1:5173"}),
		AdminAPIKeys:                getEnvAsCSV("ADMIN_API_KEYS", nil),
		DatabaseURL:                 getEnvOrDefault("DATABASE_URL", ""),
		DBMaxOpenConns:              getEnvAsInt("DB_MAX_OPEN_CONNS", 10),
		DBMaxIdleConns:              getEnvAsInt("DB_MAX_IDLE_CONNS", 5),
		DBConnMaxLifetime:           getEnvAsDuration("DB_CONN_MAX_LIFETIME", 30*time.Minute),
		DBConnMaxIdleTime:           getEnvAsDuration("DB_CONN_MAX_IDLE_TIME", 10*time.Minute),
		AliExpressAppKey:            getEnvOrDefault("ALIEXPRESS_APP_KEY", ""),
		AliExpressAppSecret:         getEnvOrDefault("ALIEXPRESS_APP_SECRET", ""),
		AliExpressDSAppKey:          getEnvOrDefault("ALIEXPRESS_DS_APP_KEY", ""),
		AliExpressDSAppSecret:       getEnvOrDefault("ALIEXPRESS_DS_APP_SECRET", ""),
		PriceUpdateScheduleEnabled:  getEnvAsBool("PRICE_UPDATE_SCHEDULE_ENABLED", false),
		PriceUpdateScheduleInterval: getEnvAsDuration("PRICE_UPDATE_SCHEDULE_INTERVAL", 24*time.Hour),
		SKUEnrichMinDelay:           getEnvAsDuration("SKU_ENRICH_MIN_DELAY", 4*time.Second),
		SKUEnrichMaxDelay:           getEnvAsDuration("SKU_ENRICH_MAX_DELAY", 7*time.Second),
		SKUSnapshotMinDelay:         getEnvAsDuration("SKU_SNAPSHOT_MIN_DELAY", 3*time.Second),
		SKUSnapshotMaxDelay:         getEnvAsDuration("SKU_SNAPSHOT_MAX_DELAY", 5*time.Second),
		TokenRefreshEnabled:         getEnvAsBool("TOKEN_REFRESH_SCHEDULE_ENABLED", false),
		TokenRefreshInterval:        getEnvAsDuration("TOKEN_REFRESH_SCHEDULE_INTERVAL", 12*time.Hour),
		HotProductScheduleEnabled:   getEnvAsBool("HOT_PRODUCT_SCHEDULE_ENABLED", false),
		HotProductScheduleInterval:  getEnvAsDuration("HOT_PRODUCT_SCHEDULE_INTERVAL", 24*time.Hour),
		HotProductSnapshotStagger:   getEnvAsDuration("HOT_PRODUCT_SNAPSHOT_STAGGER", 20*time.Minute),
		SessionCleanupEnabled:       getEnvAsBool("SESSION_CLEANUP_SCHEDULE_ENABLED", false),
		SessionCleanupInterval:      getEnvAsDuration("SESSION_CLEANUP_SCHEDULE_INTERVAL", 24*time.Hour),
		SessionCleanupRetentionDays: getEnvAsInt("SESSION_CLEANUP_RETENTION_DAYS", 90),
	}
}

func (c Config) DSN() string {
	return c.DatabaseURL
}

func getEnvOrDefault(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return defaultValue
	}
	return parsed
}

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	parsed, err := time.ParseDuration(value)
	if err != nil {
		return defaultValue
	}
	return parsed
}

func getEnvAsInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return parsed
}

func getEnvAsCSV(key string, defaultValue []string) []string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return defaultValue
	}

	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		result = append(result, part)
	}
	if len(result) == 0 {
		return defaultValue
	}
	return result
}
