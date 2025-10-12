package config

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/rohitxdev/go-api/util"
)

// This is set at build-time.
var BuildInfoBase64 string

type BuildConfig struct {
	AppName        string    `json:"app_name" validate:"required"`
	AppVersion     string    `json:"app_version" validate:"required"`
	BuildType      string    `json:"build_type" validate:"required"`
	BuildTimestamp time.Time `json:"build_timestamp" validate:"required"`
}

type RuntimeConfig struct {
	AppEnv         appEnv   `json:"app_env" validate:"required,oneof=test development staging production" env:"APP_ENV"`
	TmpDir         string   `json:"tmp_dir" validate:"required,dir" env:"TMP_DIR"`
	PostgresURL    string   `json:"postgres_url" validate:"required,url" env:"POSTGRES_URL"`
	HTTPHost       string   `json:"http_host" validate:"required" env:"HTTP_HOST"`
	HTTPPort       string   `json:"http_port" validate:"required" env:"HTTP_PORT"`
	AllowedOrigins []string `json:"allowed_origins" validate:"required"`
	Debug          bool     `json:"debug" env:"DEBUG"`
}

var Config struct {
	BuildConfig
	RuntimeConfig
}

func fetchRuntimeConfig() ([]byte, error) {
	url := os.Getenv("SECRETS_URL")
	token := os.Getenv("SECRETS_TOKEN")
	if url == "" {
		return []byte("{}"), nil
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if token != "" {
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make GET request: %w", err)
	}
	defer func() {
		if derr := res.Body.Close(); derr != nil && err == nil {
			err = fmt.Errorf("failed to close response body: %w", derr)
		}
	}()

	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", res.StatusCode, string(body))
	}

	b, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return b, nil
}

func loadConfig() error {
	decodedBuildInfo, err := base64.StdEncoding.DecodeString(BuildInfoBase64)
	if err != nil {
		return fmt.Errorf("failed to decode build info base64 string: %w", err)
	}
	if err := json.Unmarshal(decodedBuildInfo, &Config.BuildConfig); err != nil {
		return fmt.Errorf("failed to unmarshal build info: %w", err)
	}

	if err := env.Parse(&Config); err != nil {
		return fmt.Errorf("failed to parse env as config: %w", err)
	}

	Config.AllowedOrigins = strings.Split(",", os.Getenv("ALLOWED_ORIGINS"))

	b, err := fetchRuntimeConfig()
	if err != nil {
		return fmt.Errorf("failed to fetch config: %w", err)
	}
	if err = json.Unmarshal(b, &Config.RuntimeConfig); err != nil {
		return fmt.Errorf("failed to unmarshal fetched config: %w", err)
	}

	if err := util.Validate.Struct(Config); err != nil {
		return fmt.Errorf("failed to validate config: %w", err)
	}
	return nil
}

func init() {
	if err := loadConfig(); err != nil {
		panic(fmt.Errorf("failed to load config: %w", err))
	}

	go func() {
		interval := 5 * time.Minute
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			if err := loadConfig(); err != nil {
				slog.Error("failed to refetch config", slog.Any("error", err))
			}
		}
	}()
}
