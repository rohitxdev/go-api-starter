package handler

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/go-playground/validator"
	"github.com/goccy/go-json"
	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/echoprometheus"
	"github.com/labstack/echo-contrib/pprof"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rohitxdev/go-api-starter/docs"
	"github.com/rohitxdev/go-api-starter/internal/config"
	"github.com/rohitxdev/go-api-starter/pkg/id"
	"github.com/rohitxdev/go-api-starter/pkg/repo"
	echoSwagger "github.com/swaggo/echo-swagger"
	"golang.org/x/time/rate"
)

// Custom view renderer

type echoTemplate struct {
	templates *template.Template
}

func (t echoTemplate) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

// Custom request validator

type echoValidator struct {
	validator *validator.Validate
}

func (v echoValidator) Validate(i any) error {
	if err := v.validator.Struct(i); err != nil {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, err)
	}
	return nil
}

// Custom JSON serializer & deserializer

type echoJSONSerializer struct{}

func (s echoJSONSerializer) Serialize(c echo.Context, i interface{}, indent string) error {
	enc := json.NewEncoder(c.Response())
	if indent != "" {
		enc.SetIndent("", indent)
	}
	return enc.Encode(i)
}

func (s echoJSONSerializer) Deserialize(c echo.Context, i interface{}) error {
	err := json.NewDecoder(c.Request().Body).Decode(i)
	if ute, ok := err.(*json.UnmarshalTypeError); ok {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Unmarshal type error: expected=%v, got=%v, field=%v, offset=%v", ute.Type, ute.Value, ute.Field, ute.Offset)).SetInternal(err)
	} else if se, ok := err.(*json.SyntaxError); ok {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Syntax error: offset=%v, error=%v", se.Offset, se.Error())).SetInternal(err)
	}
	return err
}

// @title Starter code API
// @version 1.0
// @description This is a starter code API.

func New(h *handler) (*echo.Echo, error) {
	docs.SwaggerInfo.Host = h.config.Host + ":" + h.config.Port

	e := echo.New()

	e.HideBanner = true
	e.HidePort = true

	e.JSONSerializer = echoJSONSerializer{}

	templates, err := template.ParseFS(h.fileSystem, "web/templates/**/*.tmpl")
	if err != nil {
		return nil, fmt.Errorf("could not parse templates: %w", err)
	}
	e.Renderer = echoTemplate{
		templates: templates,
	}

	e.Validator = echoValidator{
		validator: validator.New(),
	}

	e.IPExtractor = echo.ExtractIPFromXFFHeader(
		echo.TrustLoopback(false),   // e.g. ipv4 start with 127.
		echo.TrustLinkLocal(false),  // e.g. ipv4 start with 169.254
		echo.TrustPrivateNet(false), // e.g. ipv4 start with 10. or 192.168
	)

	e.Pre(middleware.CSRF())

	e.Pre(middleware.StaticWithConfig(middleware.StaticConfig{
		Root:       "web",
		Filesystem: http.FS(h.fileSystem),
	}))

	e.Pre(middleware.Secure())

	e.Pre(middleware.CORSWithConfig(middleware.CORSConfig{AllowOrigins: h.config.AllowedOrigins}))

	e.Pre(middleware.TimeoutWithConfig(middleware.TimeoutConfig{
		Timeout: 5 * time.Second, Skipper: func(c echo.Context) bool {
			return strings.HasPrefix(c.Request().URL.Path, "/debug/pprof")
		},
	}))

	e.Pre(middleware.RequestIDWithConfig(middleware.RequestIDConfig{
		Generator: func() string {
			return id.New(id.Request)
		},
	}))

	if h.config.RateLimitPerMinute > 0 {
		e.Pre(middleware.RateLimiterWithConfig(middleware.RateLimiterConfig{
			Store: middleware.NewRateLimiterMemoryStoreWithConfig(middleware.RateLimiterMemoryStoreConfig{
				Rate:      rate.Limit(h.config.RateLimitPerMinute),
				ExpiresIn: time.Minute,
			})}))
	}

	e.Use(session.Middleware(sessions.NewCookieStore([]byte(h.config.SessionSecret))))

	// Gzip compression & decompression

	e.Use(middleware.GzipWithConfig(middleware.GzipConfig{Skipper: func(c echo.Context) bool {
		return !strings.Contains(c.Request().Header.Get("Accept-Encoding"), "gzip") || strings.HasPrefix(c.Path(), "/metrics")
	}}))

	e.Pre(middleware.Decompress())

	pprof.Register(e)

	e.Pre(middleware.RecoverWithConfig(middleware.RecoverConfig{
		LogErrorFunc: func(c echo.Context, err error, stack []byte) error {
			slog.ErrorContext(
				c.Request().Context(),
				"panic recover",
				slog.Any("error", err),
				slog.Any("stack", string(stack)),
			)
			return nil
		}},
	))

	e.Pre(middleware.RemoveTrailingSlash())

	// Prometheus metrics
	e.Use(echoprometheus.NewMiddleware("api"))
	e.GET("/metrics", echoprometheus.NewHandler())

	host, err := os.Hostname()
	if err != nil {
		return nil, fmt.Errorf("could not get host name: %w", err)
	}

	e.Pre(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogRequestID:     true,
		LogRemoteIP:      true,
		LogProtocol:      true,
		LogURI:           true,
		LogMethod:        true,
		LogStatus:        true,
		LogLatency:       true,
		LogResponseSize:  true,
		LogReferer:       true,
		LogUserAgent:     true,
		LogError:         true,
		LogContentLength: true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			var userId string
			user, ok := c.Get("user").(*repo.User)
			if ok && (user != nil) {
				userId = user.Id
			}

			slog.InfoContext(
				c.Request().Context(),
				"http request",
				slog.Group("request",
					slog.String("id", v.RequestID),
					slog.String("host", host),
					slog.String("clientIp", v.RemoteIP),
					slog.String("protocol", v.Protocol),
					slog.String("uri", v.URI),
					slog.String("method", v.Method),
					slog.String("referer", v.Referer),
					slog.String("userAgent", v.UserAgent),
					slog.String("contentLength", v.ContentLength),
					slog.Duration("durationMs", time.Duration(v.Latency.Milliseconds())),
				),
				slog.Group("response",
					slog.Int("status", v.Status),
					slog.Int64("sizeBytes", v.ResponseSize),
				),
				slog.String("userId", userId),
				slog.Any("error", v.Error),
			)

			return nil
		},
	}))

	// Routes

	e.GET("/swagger/*", echoSwagger.EchoWrapHandler(func(c *echoSwagger.Config) {
		c.SyntaxHighlight = true
	}))

	e.GET("/", func(c echo.Context) error {
		data := echo.Map{
			"buildId": config.BuildId,
			"appEnv":  h.config.AppEnv,
			"host":    host,
		}
		switch accepts(c) {
		case "text/html":
			return c.Render(http.StatusOK, "home.tmpl", data)
		default:
			return c.JSON(http.StatusOK, data)
		}
	})

	e.GET("/ping", h.Ping)

	e.GET("/_", h.AdminRoute, h.protected(RoleAdmin))

	e.GET("/config", h.GetConfig)

	e.GET("/files/:file_name", h.GetFile)

	v1 := e.Group("/v1")
	{
		auth := v1.Group("/auth")
		{
			auth.POST("/sign-up", h.SignUp)
			auth.POST("/log-in", h.LogIn)
			auth.POST("/log-out", h.LogOut)
			auth.POST("/change-password", h.ChangePassword)
		}
	}

	return e, nil
}
