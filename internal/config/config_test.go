package config

import (
	"os"
	"testing"
)

func setRequiredEnv(t *testing.T) {
	t.Helper()
	t.Setenv("MONGO_URI", "mongodb://localhost:27017")
	t.Setenv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/")
}

func TestLoad_DefaultValues(t *testing.T) {
	setRequiredEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Port != "8083" {
		t.Errorf("expected default port 8083, got %s", cfg.Port)
	}
	if cfg.MongoDB != "reports" {
		t.Errorf("expected default mongo db reports, got %s", cfg.MongoDB)
	}
	if cfg.ReportQueue != "report.queue" {
		t.Errorf("expected default report queue, got %s", cfg.ReportQueue)
	}
	if cfg.ReportTopic != "report.topic" {
		t.Errorf("expected default report topic, got %s", cfg.ReportTopic)
	}
}

func TestLoad_CustomValues(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("PORT", "9090")
	t.Setenv("MONGO_DB", "custom_reports")
	t.Setenv("REPORT_QUEUE", "custom.report.queue")
	t.Setenv("REPORT_TOPIC", "custom.report.topic")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Port != "9090" {
		t.Errorf("expected custom port, got %s", cfg.Port)
	}
	if cfg.MongoDB != "custom_reports" {
		t.Errorf("expected custom mongo db, got %s", cfg.MongoDB)
	}
	if cfg.ReportQueue != "custom.report.queue" {
		t.Errorf("expected custom report queue, got %s", cfg.ReportQueue)
	}
	if cfg.ReportTopic != "custom.report.topic" {
		t.Errorf("expected custom report topic, got %s", cfg.ReportTopic)
	}
}

func TestRequireEnv_PanicsWhenMissing(t *testing.T) {
	const key = "MISSING_REPORT_SERVICE_ENV"
	os.Unsetenv(key)

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic")
		}
	}()

	requireEnv(key)
}

func TestGetEnv(t *testing.T) {
	const key = "REPORT_SERVICE_OPTIONAL_ENV"
	os.Unsetenv(key)

	if got := getEnv(key, "fallback"); got != "fallback" {
		t.Errorf("expected fallback, got %s", got)
	}

	t.Setenv(key, "custom")
	if got := getEnv(key, "fallback"); got != "custom" {
		t.Errorf("expected custom, got %s", got)
	}
}
