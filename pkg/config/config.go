package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	RestartInterval   time.Duration
	SleepBeforeRestart time.Duration
	NumWatchers       int
	LabelSelector     string
	LogLevel          string
}

func Load() *Config {
	restartInterval := getEnvDuration("RESTART_INTERVAL", 30*time.Minute)
	sleepBeforeRestart := getEnvDuration("SLEEP_BEFORE_RESTART", 5*time.Second)
	numWatchers := getEnvInt("NUM_WATCHERS", 3)
	labelSelector := getEnvString("LABEL_SELECTOR", "")
	logLevel := getEnvString("LOG_LEVEL", "info")

	return &Config{
		RestartInterval:   restartInterval,
		SleepBeforeRestart: sleepBeforeRestart,
		NumWatchers:       numWatchers,
		LabelSelector:     labelSelector,
		LogLevel:          logLevel,
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