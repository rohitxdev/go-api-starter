package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/go-playground/validator/v10"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const (
	envDevelopment = "development"
	envProduction  = "production"
)

// This is set at build time.
var BuildId string

type Server struct {
	*Client
	Address            string         `json:"address" validate:"required"`
	GoogleOAuth2Config *oauth2.Config `json:"googleOAuth2Config"`
	Host               string         `json:"host" validate:"required,ip"`
	Port               string         `json:"port" validate:"required,gte=0"`
	SessionSecret      string         `json:"sessionSecret" validate:"required"`
	DatabaseUrl        string         `json:"databaseUrl" validate:"required"`
	SmtpHost           string         `json:"smtpHost" validate:"required"`
	SmtpUsername       string         `json:"smtpUsername" validate:"required"`
	SmtpPassword       string         `json:"smtpPassword" validate:"required"`
	SenderEmail        string         `json:"senderEmail" validate:"required"`
	S3BucketName       string         `json:"s3BucketName"`
	S3Endpoint         string         `json:"s3Endpoint"`
	S3DefaultRegion    string         `json:"s3DefaultRegion"`
	AwsAccessKeyId     string         `json:"awsAccessKeyId"`
	AwsAccessKeySecret string         `json:"awsAccessKeySecret"`
	GoogleClientId     string         `json:"googleClientId"`
	GoogleClientSecret string         `json:"googleClientSecret"`
	JwtSecret          string         `json:"jwtSecret" validate:"required"`
	AllowedOrigins     []string       `json:"allowedOrigins" validate:"required"`
	ShutdownTimeout    time.Duration  `json:"shutdownTimeout" validate:"required"`
	RateLimitPerMinute int            `json:"rateLimitPerMinute" validate:"required"`
	SmtpPort           int            `json:"smtpPort" validate:"required"`
}

type Client struct {
	Env   string `json:"env" validate:"required,oneof=development production"`
	IsDev bool   `json:"isDev"`
}

func Load() (*Server, error) {
	m := map[string]any{
		"env":  os.Getenv("ENV"),
		"host": os.Getenv("HOST"),
		"port": os.Getenv("PORT"),
	}

	if secretsFile := os.Getenv("SECRETS_FILE"); secretsFile != "" {
		data, err := os.ReadFile(secretsFile)
		if err != nil {
			return nil, fmt.Errorf("could not read secrets file: %w", err)
		}
		if err = json.Unmarshal(data, &m); err != nil {
			return nil, fmt.Errorf("could not unmarshal secrets file: %w", err)
		}
	} else {
		if secretsJson := os.Getenv("SECRETS_JSON"); secretsJson != "" {
			if err := json.Unmarshal([]byte(secretsJson), &m); err != nil {
				return nil, fmt.Errorf("could not unmarshal secrets json: %w", err)
			}
		}
	}

	var errList []error

	if m["accessTokenExpiresIn"] != nil {
		accessTokenExpiresIn, err := time.ParseDuration(m["accessTokenExpiresIn"].(string))
		if err != nil {
			errList = append(errList, fmt.Errorf("could not parse access token expiration duration: %w", err))
		}
		m["accessTokenExpiresIn"] = accessTokenExpiresIn
	}

	if m["refreshTokenExpiresIn"] != nil {
		refreshTokenExpiresIn, err := time.ParseDuration(m["refreshTokenExpiresIn"].(string))
		if err != nil {
			errList = append(errList, fmt.Errorf("could not parse refresh token expiration duration: %w", err))
		}
		m["refreshTokenExpiresIn"] = refreshTokenExpiresIn
	}

	if m["shutdownTimeout"] != nil {
		shutdownTimeout, err := time.ParseDuration(m["shutdownTimeout"].(string))
		if err != nil {
			errList = append(errList, fmt.Errorf("could not parse shutdown timeout: %w", err))
		}
		m["shutdownTimeout"] = shutdownTimeout
	}

	if len(errList) > 0 {
		return nil, errors.Join(errList...)
	}

	data, err := json.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("could not marshal config: %w", err)
	}

	var c Server
	if err = json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("could not unmarshal config: %w", err)
	}

	c.Address = net.JoinHostPort(c.Host, c.Port)
	c.GoogleOAuth2Config = &oauth2.Config{
		ClientID:     c.GoogleClientId,
		ClientSecret: c.GoogleClientSecret,
		Endpoint:     google.Endpoint,
		RedirectURL:  fmt.Sprintf("https://%s/v1/auth/oauth2/callback/google", c.Host+":"+c.Port),
		Scopes:       []string{"openid email", "openid profile"},
	}
	c.IsDev = c.Env != envProduction

	if err = validator.New().Struct(c); err != nil {
		return nil, fmt.Errorf("could not validate config: %w", err)
	}

	return &c, err
}
