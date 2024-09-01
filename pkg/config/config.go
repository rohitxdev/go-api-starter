package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/joho/godotenv"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// This is set at build time. Rest are set at run time.
var BuildInfo string

type Server struct {
	GoogleOAuth2Config    *oauth2.Config
	GitHubOAuth2Config    *oauth2.Config
	BuildInfo             string `validate:"required"`
	Host                  string `validate:"required,ip"`
	Port                  string `validate:"required,gte=0"`
	JwtSecret             string `validate:"required"`
	Env                   string `validate:"required,oneof=development production"`
	DatabaseUrl           string `validate:"required"`
	SmtpHost              string `validate:"required"`
	SmtpUsername          string `validate:"required"`
	SmtpPassword          string `validate:"required"`
	S3BucketName          string
	S3Endpoint            string
	S3DefaultRegion       string
	AwsAccessKeyId        string
	AwsAccessKeySecret    string
	GoogleClientId        string
	GoogleClientSecret    string
	AccessTokenExpiresIn  time.Duration `validate:"required"`
	RefreshTokenExpiresIn time.Duration `validate:"required"`
	ShutdownTimeout       time.Duration `validate:"required"`
	RateLimitPerMinute    int           `validate:"required"`
	SmtpPort              int           `validate:"required"`
	IsDev                 bool
}

type Client struct {
	Env string `json:"env" validate:"required,oneof=development production"`
}

func Load(envFilePath string) (*Server, error) {
	if err := godotenv.Load(envFilePath); err != nil {
		fmt.Println("warning: could not load config file: " + err.Error())
	}

	accessTokenExpiresIn, err := time.ParseDuration(os.Getenv("ACCESS_TOKEN_EXPIRES_IN"))
	if err != nil {
		return nil, errors.Join(errors.New("parse access token expiration duration"), err)
	}

	refreshTokenExpiresIn, err := time.ParseDuration(os.Getenv("REFRESH_TOKEN_EXPIRES_IN"))
	if err != nil {
		return nil, errors.Join(errors.New("parse refresh token expiration duration"), err)
	}

	smtpPort, err := strconv.ParseInt(os.Getenv("SMTP_PORT"), 10, 16)
	if err != nil {
		return nil, errors.Join(errors.New("parse SMTP port"), err)
	}

	shutdownTimeout, err := time.ParseDuration(os.Getenv("SHUTDOWN_TIMEOUT"))
	if err != nil {
		return nil, errors.Join(errors.New("parse shutdown timeout"), err)
	}

	rateLimitPerMinute, err := strconv.ParseInt(os.Getenv("RATE_LIMIT_PER_MINUTE"), 10, 8)
	if err != nil {
		return nil, errors.Join(errors.New("parse rate limit"), err)
	}

	c := Server{
		BuildInfo:             BuildInfo,
		Env:                   os.Getenv("ENV"),
		Host:                  os.Getenv("HOST"),
		Port:                  os.Getenv("PORT"),
		JwtSecret:             os.Getenv("JWT_SECRET"),
		DatabaseUrl:           os.Getenv("DATABASE_URL"),
		SmtpHost:              os.Getenv("SMTP_HOST"),
		SmtpUsername:          os.Getenv("SMTP_USERNAME"),
		SmtpPassword:          os.Getenv("SMTP_PASSWORD"),
		S3BucketName:          os.Getenv("S3_BUCKET_NAME"),
		S3Endpoint:            os.Getenv("S3_ENDPOINT"),
		S3DefaultRegion:       os.Getenv("S3_DEFAULT_REGION"),
		AwsAccessKeyId:        os.Getenv("AWS_ACCESS_KEY_ID"),
		AwsAccessKeySecret:    os.Getenv("AWS_ACCESS_KEY_SECRET"),
		GoogleClientId:        os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret:    os.Getenv("GOOGLE_CLIENT_SECRET"),
		AccessTokenExpiresIn:  accessTokenExpiresIn,
		RefreshTokenExpiresIn: refreshTokenExpiresIn,
		ShutdownTimeout:       shutdownTimeout,
		RateLimitPerMinute:    int(rateLimitPerMinute),

		SmtpPort: int(smtpPort),
		IsDev:    os.Getenv("APP_ENV") != "production",
	}

	if err := validator.New().Struct(c); err != nil {
		return nil, errors.Join(errors.New("validate config"), err)
	}

	if c.GoogleClientId != "" && c.GoogleClientSecret != "" {
		c.GoogleOAuth2Config = &oauth2.Config{
			ClientID:     c.GoogleClientId,
			ClientSecret: c.GoogleClientSecret,
			Endpoint:     google.Endpoint,
			RedirectURL:  "https://localhost:8443/v1/auth/oauth2/callback/google",
			Scopes:       []string{"openid email", "openid profile"},
		}
	}

	return &c, nil
}