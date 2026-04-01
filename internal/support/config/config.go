package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port                        string
	DatabaseURL                 string
	AliExpressAppKey            string
	AliExpressAppSecret         string
	AliExpressDSAppKey          string
	AliExpressDSAppSecret       string
	PriceUpdateScheduleEnabled  bool
	PriceUpdateScheduleInterval time.Duration
	TokenRefreshEnabled         bool
	TokenRefreshInterval        time.Duration
}

func Load() Config {
	return Config{
		Port:                        getEnvOrDefault("PORT", "8700"),
		DatabaseURL:                 getEnvOrDefault("DATABASE_URL", ""),
		AliExpressAppKey:            getEnvOrDefault("ALIEXPRESS_APP_KEY", ""),
		AliExpressAppSecret:         getEnvOrDefault("ALIEXPRESS_APP_SECRET", ""),
		AliExpressDSAppKey:          getEnvOrDefault("ALIEXPRESS_DS_APP_KEY", ""),
		AliExpressDSAppSecret:       getEnvOrDefault("ALIEXPRESS_DS_APP_SECRET", ""),
		PriceUpdateScheduleEnabled:  getEnvAsBool("PRICE_UPDATE_SCHEDULE_ENABLED", false),
		PriceUpdateScheduleInterval: getEnvAsDuration("PRICE_UPDATE_SCHEDULE_INTERVAL", 24*time.Hour),
		TokenRefreshEnabled:         getEnvAsBool("TOKEN_REFRESH_SCHEDULE_ENABLED", false),
		TokenRefreshInterval:        getEnvAsDuration("TOKEN_REFRESH_SCHEDULE_INTERVAL", 12*time.Hour),
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
