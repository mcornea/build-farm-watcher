package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	RestartInterval      time.Duration
	SleepBeforeRestart   time.Duration
	NumWatchers          int
	LabelSelector        string
	LogLevel             string
	SecretsListInterval  time.Duration
	EnableSecretsListing bool
}

func Load() *Config {
	restartInterval := getEnvDuration("RESTART_INTERVAL", 30*time.Minute)
	sleepBeforeRestart := getEnvDuration("SLEEP_BEFORE_RESTART", 5*time.Second)
	numWatchers := getEnvInt("NUM_WATCHERS", 3)
	labelSelector := getEnvString("LABEL_SELECTOR", "")
	logLevel := getEnvString("LOG_LEVEL", "info")
	secretsListInterval := getEnvDuration("SECRETS_LIST_INTERVAL", 10*time.Second)
	enableSecretsListing := getEnvBool("ENABLE_SECRETS_LISTING", false)

	return &Config{
		RestartInterval:      restartInterval,
		SleepBeforeRestart:   sleepBeforeRestart,
		NumWatchers:          numWatchers,
		LabelSelector:        labelSelector,
		LogLevel:             logLevel,
		SecretsListInterval:  secretsListInterval,
		EnableSecretsListing: enableSecretsListing,
	}
}

func getEnvString(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}