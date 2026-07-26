package config

import (
	"os"
	"time"
)

type Config struct {
	Server         ServerConfig
	Database       DatabaseConfig
	Redis          RedisConfig
	OIDC           OIDCConfig
	Policy         PolicyConfig
	Agent          AgentConfig
	Signing        SigningConfig
}

type ServerConfig struct {
	Host         string
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

type DatabaseConfig struct {
	Host            string
	Port            string
	User            string
	Password        string
	Name            string
	MaxConns        int32
	MinConns        int32
	MaxConnLifetime time.Duration
}

func (d *DatabaseConfig) URL() string {
	return "postgres://" + d.User + ":" + d.Password + "@" + d.Host + ":" + d.Port + "/" + d.Name + "?sslmode=disable"
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

func (r *RedisConfig) Addr() string {
	return r.Host + ":" + r.Port
}

type OIDCConfig struct {
	Issuer   string
	Audience string
}

type PolicyConfig struct {
	BundlePath string
	CacheTTL   time.Duration
}

type AgentConfig struct {
	PollInterval time.Duration
	Timeout      time.Duration
}

type SigningConfig struct {
	FulcioURL   string
	RekorURL    string
	OIDCIssuer  string
}

func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Host:         getEnv("SERVER_HOST", "0.0.0.0"),
			Port:         getEnv("SERVER_PORT", "8080"),
			ReadTimeout:  getDurationEnv("SERVER_READ_TIMEOUT", 10*time.Second),
			WriteTimeout: getDurationEnv("SERVER_WRITE_TIMEOUT", 30*time.Second),
			IdleTimeout:  getDurationEnv("SERVER_IDLE_TIMEOUT", 120*time.Second),
		},
		Database: DatabaseConfig{
			Host:            getEnv("DB_HOST", "localhost"),
			Port:            getEnv("DB_PORT", "5432"),
			User:            getEnv("DB_USER", "deploymate"),
			Password:        getEnv("DB_PASSWORD", "deploymate"),
			Name:            getEnv("DB_NAME", "deploymate"),
			MaxConns:        25,
			MinConns:        5,
			MaxConnLifetime: 1 * time.Hour,
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       0,
		},
		OIDC: OIDCConfig{
			Issuer:   getEnv("OIDC_ISSUER", "https://token.actions.githubusercontent.com"),
			Audience: getEnv("OIDC_AUDIENCE", "deploymate"),
		},
		Policy: PolicyConfig{
			BundlePath: getEnv("POLICY_BUNDLE_PATH", "/var/lib/deploymate/policies"),
			CacheTTL:   5 * time.Minute,
		},
		Agent: AgentConfig{
			PollInterval: 10 * time.Second,
			Timeout:      30 * time.Second,
		},
		Signing: SigningConfig{
			FulcioURL:  getEnv("FULCIO_URL", "https://fulcio.sigstore.dev"),
			RekorURL:   getEnv("REKOR_URL", "https://rekor.sigstore.dev"),
			OIDCIssuer: getEnv("SIGNING_OIDC_ISSUER", "https://token.actions.githubusercontent.com"),
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if d, err := time.ParseDuration(value); err == nil {
			return d
		}
	}
	return defaultValue
}
