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

// These variables are set by the build process.
var (
	AppName    string
	AppVersion string
	BuildType  string
)

type Config struct {
	AppName    string `json:"appName"`
	AppVersion string `json:"appVersion"`
	BuildType  string `json:"buildType"`
	Env        string `json:"env" validate:"required,oneof=development production"`
	Host       string `json:"host" validate:"required,ip"`
	Port       string `json:"port" validate:"required,gte=0"`
	// DatabaseURL is the connection string of the database.
	DatabaseURL  string `json:"databaseUrl" validate:"required"`
	SMTPHost     string `json:"smtpHost" validate:"required"`
	SMTPUsername string `json:"smtpUsername" validate:"required"`
	SMTPPassword string `json:"smtpPassword" validate:"required"`
	// SenderEmail is the email address from which emails will be sent.
	SenderEmail        string         `json:"senderEmail" validate:"required"`
	S3BucketName       string         `json:"s3BucketName"`
	S3Endpoint         string         `json:"s3Endpoint"`
	S3DefaultRegion    string         `json:"s3DefaultRegion"`
	AWSAccessKeyID     string         `json:"awsAccessKeyId"`
	AWSAccessKeySecret string         `json:"awsAccessKeySecret"`
	GoogleOAuth2Config *oauth2.Config `json:"googleOAuth2Config"`
	// GoogleClientID is the client ID for Google OAuth2 authentication.
	GoogleClientID     string `json:"googleClientId"`
	GoogleClientSecret string `json:"googleClientSecret"`
	// SessionSecret is the secret key used to sign session cookies.
	SessionSecret string `json:"sessionSecret" validate:"required"`
	// JWTSecret is the secret key used to sign JWT tokens.
	JWTSecret string `json:"jwtSecret" validate:"required"`
	// AllowedOrigins is a list of origins that are allowed to access the API.
	AllowedOrigins  []string      `json:"allowedOrigins"`
	SessionDuration time.Duration `json:"sessionDuration" validate:"required"`
	// LogInTokenExpiresIn is the duration after which the log-in token in email will expire.
	LogInTokenExpiresIn time.Duration `json:"logInTokenExpiresIn" validate:"required"`
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
	if m["sessionDuration"], err = time.ParseDuration(m["sessionDuration"].(string)); err != nil {
		errList = append(errList, fmt.Errorf("Failed to parse session duration: %w", err))
	}
	if m["logInTokenExpiresIn"], err = time.ParseDuration(m["logInTokenExpiresIn"].(string)); err != nil {
		errList = append(errList, fmt.Errorf("Failed to parse log in token expires in: %w", err))
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
	cfg.AppVersion = AppVersion
	cfg.BuildType = BuildType
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
