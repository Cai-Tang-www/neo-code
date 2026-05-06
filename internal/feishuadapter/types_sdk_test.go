package feishuadapter

import (
	"testing"
	"time"
)

func TestConfigValidateSDKDoesNotRequireWebhookFields(t *testing.T) {
	cfg := Config{
		IngressMode:         IngressModeSDK,
		AppID:               "app",
		AppSecret:           "secret",
		RequestTimeout:      3 * time.Second,
		IdempotencyTTL:      time.Minute,
		ReconnectBackoffMin: time.Second,
		ReconnectBackoffMax: 2 * time.Second,
		RebindInterval:      3 * time.Second,
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("validate sdk mode: %v", err)
	}
}

func TestConfigValidateRejectsInvalidIngressMode(t *testing.T) {
	cfg := Config{
		IngressMode:         "other",
		AppID:               "app",
		AppSecret:           "secret",
		RequestTimeout:      time.Second,
		IdempotencyTTL:      time.Minute,
		ReconnectBackoffMin: time.Second,
		ReconnectBackoffMax: time.Second,
		RebindInterval:      time.Second,
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected invalid ingress mode error")
	}
}
