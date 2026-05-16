package config

import (
	"fmt"
	"os"
)

type Config struct {
	Port        string
	MongoURI    string
	MongoDB     string
	RabbitMQURL string
	ReportQueue string
	ReportTopic string
}

func Load() (*Config, error) {
	return &Config{
		Port:        getEnv("PORT", "8083"),
		MongoURI:    requireEnv("MONGO_URI"),
		MongoDB:     getEnv("MONGO_DB", "reports"),
		RabbitMQURL: requireEnv("RABBITMQ_URL"),
		ReportQueue: getEnv("REPORT_QUEUE", "report.queue"),
		ReportTopic: getEnv("REPORT_TOPIC", "report.topic"),
	}, nil
}

func requireEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic(fmt.Sprintf("required env var %s is not set", key))
	}
	return v
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
