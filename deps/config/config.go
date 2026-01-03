package config

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/rohitxdev/go-api/util"
)

// This must be set at build-time.
var (
	BuildInfoBase64 string
)

var (
	ErrBuildInfoNotSet     = errors.New("build info is not set")
	ErrConfigNil           = errors.New("config is nil")
	ErrStoreNotInitialized = errors.New("config store not initialized")
)

type Environment = string

const (
	EnvTest        Environment = "testing"
	EnvDevelopment Environment = "development"
	EnvStaging     Environment = "staging"
	EnvProduction  Environment = "production"
)

type Build struct {
	AppName        string    `json:"app_name" validate:"required"`
	AppVersion     string    `json:"app_version" validate:"required"`
	BuildType      string    `json:"build_type" validate:"required"`
	BuildTimestamp time.Time `json:"build_timestamp" validate:"required"`
}

type Runtime struct {
	AllowedOrigins          []string      `json:"allowed_origins" validate:"required,dive,min=1" env:"ALLOWED_ORIGINS"`
	AppEnv                  Environment   `json:"app_env" validate:"required,oneof=testing development staging production" env:"APP_ENV"`
	TmpDir                  string        `json:"tmp_dir" validate:"required,dir" env:"TMP_DIR"`
	HTTPHost                string        `json:"http_host" validate:"required" env:"HTTP_HOST"`
	HTTPPort                string        `json:"http_port" validate:"required" env:"HTTP_PORT"`
	EmailFromAddress        string        `json:"email_from_address" env:"EMAIL_FROM_ADDRESS"`
	EmailFromName           string        `json:"email_from_name" env:"EMAIL_FROM_NAME"`
	SessionTTL              time.Duration `json:"session_ttl" validate:"required" env:"SESSION_TTL"`
	VerificationCodeTTL     time.Duration `json:"verification_code_ttl" validate:"required" env:"VERIFICATION_CODE_TTL"`
	MaxVerificationAttempts int           `json:"max_verification_attempts" validate:"required,min=1" env:"MAX_VERIFICATION_ATTEMPTS"`
	Debug                   bool          `json:"debug" env:"DEBUG"`
}

type Secrets struct {
	PostgresURL    string `json:"postgres_url" validate:"required,url" env:"POSTGRES_URL"`
	RedisURL       string `json:"redis_url" validate:"required,url" env:"REDIS_URL"`
	SessionSecret  string `json:"session_secret" validate:"required,len=64" env:"SESSION_SECRET"`
	DeviceIDSecret string `json:"device_id_secret" validate:"required,len=64" env:"DEVICE_ID_SECRET"`
	SMTPUsername   string `json:"smtp_username" env:"SMTP_username"`
	SMTPPassword   string `json:"smtp_password" env:"SMTP_PASSWORD"`
	SMTPHost       string `json:"smtp_host" env:"SMTP_HOST"`
	SMTPPort       int    `json:"smtp_port" env:"SMTP_PORT"`
}

type Features struct {
	EmailVerificationEnabled bool `json:"email_verification_enabled" env:"EMAIL_VERIFICATION_ENABLED"`
}

type Config struct {
	Build
	Runtime
	Secrets
	Features
}

func validateConfig(cfg *Config) error {
	if err := util.Validate.Struct(cfg); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}
	return nil
}

func loadConfig() (*Config, error) {
	if BuildInfoBase64 == "" {
		return nil, ErrBuildInfoNotSet
	}

	decoded, err := base64.StdEncoding.DecodeString(BuildInfoBase64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode build info base64 string: %w", err)
	}

	var cfg Config

	if err := json.Unmarshal(decoded, &cfg.Build); err != nil {
		return nil, fmt.Errorf("failed to unmarshal build info: %w", err)
	}

	if err := env.Parse(&cfg); err != nil {
		return nil, fmt.Errorf("failed to parse env as config: %w", err)
	}

	if err := validateConfig(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

type Store struct {
	cfg atomic.Pointer[Config]
}

func NewStore() (*Store, error) {
	cfg, err := loadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	var store Store
	store.cfg.Store(cfg)

	return &store, nil
}

func (s *Store) Get() *Config {
	return s.cfg.Load()
}

func (s *Store) Set(newCfg *Config) error {
	if newCfg == nil {
		return ErrConfigNil
	}

	val := *newCfg
	cfg := s.cfg.Load()
	if cfg == nil {
		return ErrStoreNotInitialized
	}

	val.Build = cfg.Build
	if err := validateConfig(&val); err != nil {
		return err
	}

	s.cfg.Store(&val)

	return nil
}
