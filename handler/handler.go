package handler

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"text/template"
	"time"

	"github.com/eko/gocache/lib/v4/cache"
	"github.com/go-playground/validator"
	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/echoprometheus"
	"github.com/labstack/echo-contrib/pprof"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/oklog/ulid/v2"
	"github.com/rohitxdev/go-api/assets"
	"github.com/rohitxdev/go-api/config"
	"github.com/rohitxdev/go-api/database/repository"
)

type Services struct {
	Cache *cache.Cache[string]
	Repo  *repository.Queries
}

type Handler struct {
	*Services
}

func registerRoutes(e *echo.Echo, h *Handler) {
	e.GET("/metrics", echoprometheus.NewHandler())
	e.GET("/health", h.GetHealth)
	e.GET("/config", h.GetConfig)
	e.GET("/", h.Home)

	auth := e.Group("/auth")
	{
		auth.POST("/otp/send", h.SendAuthOTP)
		auth.POST("/otp/verify", h.VerifyAuthOTP)
		auth.POST("/sign-out", h.SignOut)
	}

	users := e.Group("/users")
	{
		users.GET("/me", h.GetMe)
	}
}

func New(svc *Services) (*echo.Echo, error) {
	h := Handler{Services: svc}

	e := echo.New()
	e.JSONSerializer = jsonSerializer{}
	e.Validator = requestValidator{
		validator: validator.New(),
	}
	e.IPExtractor = echo.ExtractIPFromXFFHeader(
		echo.TrustLoopback(false),   // e.g. ipv4 start with 127.
		echo.TrustLinkLocal(false),  // e.g. ipv4 start with 169.254
		echo.TrustPrivateNet(false), // e.g. ipv4 start with 10. or 192.168
	)

	pageTemplates, err := template.ParseFS(assets.FS, "templates/pages/*.tmpl")
	if err != nil {
		return nil, fmt.Errorf("failed to parse templates: %w", err)
	}
	e.Renderer = viewRenderer{
		templates: pageTemplates,
	}

	e.HTTPErrorHandler = func(err error, c echo.Context) {
		defer func() {
			if err != nil {
				slog.Error("HTTP error response failure",
					slog.Group("request", slog.String("id", c.Response().Header().Get(echo.HeaderXRequestID))),
					slog.String("error", err.Error()),
				)
			}
		}()

		var res APIResponse
		if httpErr, ok := err.(*echo.HTTPError); ok {
			switch msg := httpErr.Message.(type) {
			case string:
				res.Message = msg
			case error:
				res.Message = msg.Error()
			default:
				res.Message = httpErr.Error()
			}
			err = c.JSON(httpErr.Code, res)
		} else {
			res.Message = MsgInternalServerError
			err = c.JSON(http.StatusInternalServerError, res)
		}
	}

	//Pre-router middlewares
	e.Pre(middleware.CSRF())

	e.Pre(middleware.Secure())

	e.Pre(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:                             cfg.AllowedOrigins,
		AllowCredentials:                         true,
		UnsafeWildcardOriginWithAllowCredentials: cfg.AppEnv != config.EnvProduction,
	}))

	e.Pre(middleware.StaticWithConfig(middleware.StaticConfig{
		Root:       "public",
		Filesystem: http.FS(assets.FS),
	}))

	e.Pre(middleware.RequestIDWithConfig(middleware.RequestIDConfig{
		Generator: ulid.Make().String,
	}))

	e.Pre(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogRequestID:    true,
		LogRemoteIP:     true,
		LogProtocol:     true,
		LogURI:          true,
		LogMethod:       true,
		LogStatus:       true,
		LogLatency:      true,
		LogResponseSize: true,
		LogReferer:      true,
		LogUserAgent:    true,
		LogError:        true,
		LogHost:         true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			status := v.Status
			var errStr string
			if httpErr, ok := v.Error.(*echo.HTTPError); ok {
				if httpErr.Code == http.StatusInternalServerError {
					errStr = httpErr.Error()
				}
			} else if v.Error != nil {
				// Due to a bug in echo, when the error is not an echo.HTTPError, even though the status code sent is 500, it's logged as 200 in this middleware.
				// We need to manually set the status code in the log to 500.
				status = http.StatusInternalServerError
			}

			var userID string
			if user, ok := c.Get("user").(*repository.User); ok && (user != nil) {
				userID = user.ID.String()
			}

			attrs := []any{
				slog.String("id", v.RequestID),
				slog.String("client_ip", v.RemoteIP),
				slog.String("protocol", v.Protocol),
				slog.String("uri", v.URI),
				slog.String("method", v.Method),
				slog.Int64("duration_ms", v.Latency.Milliseconds()),
				slog.Int64("res_bytes", v.ResponseSize),
				slog.Int("status", status),
			}
			if v.Host != "" {
				attrs = append(attrs, slog.String("host", v.Host))
			}
			if v.UserAgent != "" {
				attrs = append(attrs, slog.String("user_agent", v.UserAgent))
			}
			if v.Referer != "" {
				attrs = append(attrs, slog.String("referer", v.Referer))
			}
			if userID != "" {
				attrs = append(attrs, slog.String("user_id", userID))
			}
			if errStr != "" {
				attrs = append(attrs, slog.String("error", errStr))
			}

			slog.Info("HTTP request", attrs...)
			return nil
		},
	}))

	e.Pre(middleware.RemoveTrailingSlash())

	//Post-router middlewares
	sessionStore := sessions.NewCookieStore([]byte("auth-secret"))
	e.Use(session.Middleware(sessionStore))

	e.Use(middleware.GzipWithConfig(middleware.GzipConfig{
		Skipper: func(c echo.Context) bool {
			return !strings.Contains(c.Request().Header.Get("Accept-Encoding"), "gzip") || strings.HasPrefix(c.Path(), "/metrics")
		},
	}))

	e.Use(middleware.Decompress())

	e.Use(middleware.RecoverWithConfig(middleware.RecoverConfig{
		LogErrorFunc: func(c echo.Context, err error, stack []byte) error {
			slog.Error("http handler panic", slog.String("id", c.Response().Header().Get(echo.HeaderXRequestID)), slog.String("error", err.Error()), slog.String("stack", string(stack)))
			return nil
		}},
	))

	// This middleware causes data races, but it's not a big deal. See https://github.com/labstack/echo/issues/1761
	e.Use(middleware.TimeoutWithConfig(middleware.TimeoutConfig{
		Timeout: time.Minute,
		Skipper: func(c echo.Context) bool {
			return strings.HasPrefix(c.Path(), "/debug/pprof")
		},
	}))

	e.Use(echoprometheus.NewMiddleware("api"))

	pprof.Register(e)

	registerRoutes(e, &h)

	return e, nil
}
