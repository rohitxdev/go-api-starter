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

type Build struct {
	AppName        string    `json:"app_name" validate:"required"`
	AppVersion     string    `json:"app_version" validate:"required"`
	BuildType      string    `json:"build_type" validate:"required"`
	BuildTimestamp time.Time `json:"build_timestamp" validate:"required"`
}

type Runtime struct {
	AppEnv         Environment `json:"app_env" validate:"required,oneof=testing development staging production" env:"APP_ENV"`
	TmpDir         string      `json:"tmp_dir" validate:"required,dir" env:"TMP_DIR"`
	HTTPHost       string      `json:"http_host" validate:"required" env:"HTTP_HOST"`
	HTTPPort       string      `json:"http_port" validate:"required" env:"HTTP_PORT"`
	AllowedOrigins []string    `json:"allowed_origins" validate:"required,dive,min=1" env:"ALLOWED_ORIGINS"`
	Debug          bool        `json:"debug" env:"DEBUG"`
}

type Secrets struct {
	PostgresURL   string `json:"postgres_url" validate:"required,url" env:"POSTGRES_URL"`
	RedisURL      string `json:"redis_url" validate:"required,url" env:"REDIS_URL"`
	SessionSecret string `json:"session_secret" validate:"required,len=64" env:"SESSION_SECRET"`
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
