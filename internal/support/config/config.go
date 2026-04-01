package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port                        string
	DBHost                      string
	DBPort                      string
	DBUser                      string
	DBPassword                  string
	DBName                      string
	DBSSLMode                   string
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
		Port:                        getEnvOrDefault("PORT", "8081"),
		DBHost:                      getEnvOrDefault("DB_HOST", "localhost"),
		DBPort:                      getEnvOrDefault("DB_PORT", "5432"),
		DBUser:                      getEnvOrDefault("DB_USER", "postgres"),
		DBPassword:                  getEnvOrDefault("DB_PASSWORD", "postgres"),
		DBName:                      getEnvOrDefault("DB_NAME", "gugu"),
		DBSSLMode:                   getEnvOrDefault("DB_SSL_MODE", "disable"),
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
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.DBHost, c.DBPort, c.DBUser, c.DBPassword, c.DBName, c.DBSSLMode,
	)
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
