package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/go-playground/validator/v10"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// This is set at compile time.
var (
	AppName   string
	BuildType string
	BuildID   string
)

type Config struct {
	// AppName is the name of the application as written in go.mod file.
	AppName string `json:"appName" validate:"required"`
	// BuildType is the type of build, either "debug" or "release".
	BuildType string `json:"buildType" validate:"required"`
	// BuildID is the build ID of the application.
	BuildID string `json:"buildId" validate:"required"`
	// Env is the environment in which the server is running.
	Env string `json:"env" validate:"required,oneof=development production"`
	// GoogleOAuth2Config is the configuration for Google OAuth2 authentication.
	GoogleOAuth2Config *oauth2.Config `json:"googleOAuth2Config"`
	// Host is the hostname of the server.
	Host string `json:"host" validate:"required,ip"`
	// Port is the port of the server.
	Port string `json:"port" validate:"required,gte=0"`
	// DatabaseURL is the URL of the database.
	DatabaseURL string `json:"databaseUrl" validate:"required"`
	// SMTPHost is the host of the SMTP server.
	SMTPHost string `json:"smtpHost" validate:"required"`
	// SMTPUsername is the username for the SMTP server.
	SMTPUsername string `json:"smtpUsername" validate:"required"`
	// SMTPPassword is the password for the SMTP server.
	SMTPPassword string `json:"smtpPassword" validate:"required"`
	// SenderEmail is the email address from which emails will be sent.
	SenderEmail string `json:"senderEmail" validate:"required"`
	// S3BucketName is the name of the S3 bucket.
	S3BucketName string `json:"s3BucketName"`
	// S3Endpoint is the endpoint of the S3 server.
	S3Endpoint string `json:"s3Endpoint"`
	// S3DefaultRegion is the default region of the S3 server.
	S3DefaultRegion string `json:"s3DefaultRegion"`
	// AWSAccessKeyID is the access key ID for the S3 server.
	AWSAccessKeyID string `json:"awsAccessKeyId"`
	// AWSAccessKeySecret is the access key secret for the S3 server.
	AWSAccessKeySecret string `json:"awsAccessKeySecret"`
	// GoogleClientID is the client ID for Google OAuth2 authentication.
	GoogleClientID string `json:"googleClientId"`
	// GoogleClientSecret is the client secret for Google OAuth2 authentication.
	GoogleClientSecret string `json:"googleClientSecret"`
	// SessionSecret is the secret key used to sign session cookies.
	SessionSecret string `json:"sessionSecret" validate:"required"`
	// JWTSecret is the secret key used to sign JWT tokens.
	JWTSecret string `json:"jwtSecret" validate:"required"`
	// AllowedOrigins is a list of origins that are allowed to access the API.
	AllowedOrigins []string `json:"allowedOrigins"`
	// ShutdownTimeout is the duration after which the server will be shutdown gracefully.
	ShutdownTimeout time.Duration `json:"shutdownTimeout" validate:"required"`
	// RateLimitPerMinute is the rate limit per minute for the API.
	RateLimitPerMinute int `json:"rateLimitPerMinute" validate:"required"`
	// SMTPPort is the port of the SMTP server.
	SMTPPort int `json:"smtpPort" validate:"required"`
	// IsDev is a flag indicating whether the server is running in development mode.
	IsDev bool `json:"isDev"`
}

func Load() (*Config, error) {
	var m map[string]any
	var err error

	if secretsFile := os.Getenv("SECRETS_FILE"); secretsFile != "" {
		if m, err = loadFromFile(secretsFile); err != nil {
			return nil, fmt.Errorf("Failed to load secrets file: %w", err)
		}
	} else if secretsJSON := os.Getenv("SECRETS_JSON"); secretsJSON != "" {
		if m, err = loadFromJSON(secretsJSON); err != nil {
			return nil, fmt.Errorf("Failed to load secrets JSON: %w", err)
		}
	} else {
		return nil, errors.New("SECRETS_FILE or SECRETS_JSON must be set")
	}

	var errList []error

	m["env"] = os.Getenv("ENV")
	m["host"] = os.Getenv("HOST")
	m["port"] = os.Getenv("PORT")
	if m["shutdownTimeout"], err = time.ParseDuration(m["shutdownTimeout"].(string)); err != nil {
		errList = append(errList, fmt.Errorf("Failed to parse shutdown timeout: %w", err))
	}

	if len(errList) > 0 {
		return nil, errors.Join(errList...)
	}

	data, err := json.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("Failed to marshal config: %w", err)
	}

	var cfg Config
	if err = json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("Failed to unmarshal config: %w", err)
	}

	cfg.GoogleOAuth2Config = &oauth2.Config{
		ClientID:     cfg.GoogleClientID,
		ClientSecret: cfg.GoogleClientSecret,
		Endpoint:     google.Endpoint,
		RedirectURL:  fmt.Sprintf("https://%s/v1/auth/oauth2/callback/google", cfg.Host+":"+cfg.Port),
		Scopes:       []string{"openid email", "openid profile"},
	}
	cfg.AppName = AppName
	cfg.BuildType = BuildType
	cfg.BuildID = BuildID
	cfg.IsDev = cfg.Env != "production"

	if err = validator.New().Struct(cfg); err != nil {
		return nil, fmt.Errorf("Failed to validate config: %w", err)
	}

	return &cfg, err
}

func loadFromFile(secretsFile string) (map[string]any, error) {
	data, err := os.ReadFile(secretsFile)
	if err != nil {
		return nil, fmt.Errorf("Failed to read secrets file: %w", err)
	}
	var m map[string]any
	if err = json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("Failed to unmarshal secrets file: %w", err)
	}
	return m, nil
}

func loadFromJSON(jsonData string) (map[string]any, error) {
	var m map[string]any
	if err := json.Unmarshal([]byte(jsonData), &m); err != nil {
		return nil, fmt.Errorf("Failed to unmarshal secrets json: %w", err)
	}
	return m, nil
}
