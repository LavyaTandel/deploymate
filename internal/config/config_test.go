package config

import (
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	cfg := Load()
	if cfg == nil {
		t.Fatal("Load() returned nil")
	}
	if cfg.Server.Port == "" {
		t.Error("Server.Port is empty")
	}
	if cfg.Database.Host == "" {
		t.Error("Database.Host is empty")
	}
}

func TestDatabaseURL(t *testing.T) {
	db := DatabaseConfig{
		Host: "localhost",
		Port: "5432",
		User: "test",
		Password: "secret",
		Name: "mydb",
	}
	got := db.URL()
	want := "postgres://test:secret@localhost:5432/mydb?sslmode=disable"
	if got != want {
		t.Errorf("URL() = %q, want %q", got, want)
	}
}

func TestRedisAddr(t *testing.T) {
	r := RedisConfig{Host: "redis.local", Port: "6379"}
	got := r.Addr()
	want := "redis.local:6379"
	if got != want {
		t.Errorf("Addr() = %q, want %q", got, want)
	}
}

func TestGetEnv(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		fallback string
		envVal   string
		want     string
	}{
		{"env set", "TEST_DM_KEY", "default", "custom", "custom"},
		{"env unset", "TEST_DM_MISSING", "default", "", "default"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envVal != "" {
				os.Setenv(tt.key, tt.envVal)
				defer os.Unsetenv(tt.key)
			}
			got := getEnv(tt.key, tt.fallback)
			if got != tt.want {
				t.Errorf("getEnv() = %q, want %q", got, tt.want)
			}
		})
	}
}
