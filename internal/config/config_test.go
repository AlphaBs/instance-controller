package config

import "testing"

func TestLoad(t *testing.T) {
	t.Setenv("EC2_REGION", "ap-northeast-2")
	t.Setenv("EC2_INSTANCE_ID", "i-test")
	t.Setenv("BASIC_AUTH_USERNAME", "user")
	t.Setenv("BASIC_AUTH_PASSWORD", "secret")
	t.Setenv("PORT", "")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Port != "8080" {
		t.Fatalf("port = %q, want 8080", cfg.Port)
	}
}

func TestLoadRejectsMissingRequiredValue(t *testing.T) {
	t.Setenv("EC2_REGION", "")
	t.Setenv("AWS_REGION", "")
	t.Setenv("EC2_INSTANCE_ID", "i-test")
	t.Setenv("BASIC_AUTH_USERNAME", "user")
	t.Setenv("BASIC_AUTH_PASSWORD", "secret")

	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil, want missing EC2_REGION error")
	}
}
