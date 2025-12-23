package config

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/rohitxdev/go-api/util"
)

// This is set at build-time.
var (
	BuildInfoBase64 string
)

var (
	ErrBuildInfoNotSet = errors.New("build info is not set")
)

type BuildConfig struct {
	AppName        string    `json:"app_name" validate:"required"`
	AppVersion     string    `json:"app_version" validate:"required"`
	BuildType      string    `json:"build_type" validate:"required"`
	BuildTimestamp time.Time `json:"build_timestamp" validate:"required"`
}

type RuntimeConfig struct {
	AppEnv         Environment `json:"app_env" validate:"required,oneof=testing development staging production" env:"APP_ENV"`
	TmpDir         string      `json:"tmp_dir" validate:"required,dir" env:"TMP_DIR"`
	HTTPHost       string      `json:"http_host" validate:"required" env:"HTTP_HOST"`
	HTTPPort       string      `json:"http_port" validate:"required" env:"HTTP_PORT"`
	AllowedOrigins []string    `json:"allowed_origins" validate:"required,dive,min=1" env:"ALLOWED_ORIGINS"`
}

type Secrets struct {
	PostgresURL   string `json:"postgres_url" validate:"required,url" env:"POSTGRES_URL"`
	RedisURL      string `json:"redis_url" validate:"required,url" env:"REDIS_URL"`
	SessionSecret string `json:"session_secret" validate:"required,len=64" env:"SESSION_SECRET"`
}

type FeatureFlags struct {
	Debug bool `json:"debug" env:"DEBUG"`
}

type Config struct {
	BuildConfig
	RuntimeConfig
	Secrets
	FeatureFlags
}

func Load() (*Config, error) {
	if BuildInfoBase64 == "" {
		return nil, ErrBuildInfoNotSet
	}

	decoded, err := base64.StdEncoding.DecodeString(BuildInfoBase64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode build info base64 string: %w", err)
	}

	var cfg Config

	if err := json.Unmarshal(decoded, &cfg.BuildConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal build info: %w", err)
	}

	if err := env.Parse(&cfg); err != nil {
		return nil, fmt.Errorf("failed to parse env as config: %w", err)
	}

	if err := util.Validate.Struct(cfg); err != nil {
		return nil, fmt.Errorf("failed to validate config: %w", err)
	}

	return &cfg, nil
}
