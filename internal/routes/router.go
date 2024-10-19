package routes

import (
	"fmt"
	"html/template"
	"io"
	"log/slog"
	"net/http"
	"strings"
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
	"github.com/rohitxdev/go-api-starter/internal/id"
	"github.com/rohitxdev/go-api-starter/internal/repo"
	echoSwagger "github.com/swaggo/echo-swagger"
	"golang.org/x/time/rate"
)

// Custom view renderer
type customRenderer struct {
	templates *template.Template
}

func (t customRenderer) Render(w io.Writer, name string, data any, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

// Custom request validator
type customValidator struct {
	validator *validator.Validate
}

func (v customValidator) Validate(i any) error {
	if err := v.validator.Struct(i); err != nil {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, err)
	}
	return nil
}

// Custom JSON serializer & deserializer
type customJSONSerializer struct{}

func (s customJSONSerializer) Serialize(c echo.Context, data any, indent string) error {
	enc := json.NewEncoder(c.Response())
	if indent != "" {
		enc.SetIndent("", indent)
	}
	return enc.Encode(data)
}

func (s customJSONSerializer) Deserialize(c echo.Context, v any) error {
	err := json.NewDecoder(c.Request().Body).Decode(v)
	if ute, ok := err.(*json.UnmarshalTypeError); ok {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Unmarshal type error: expected=%v, got=%v, field=%v, offset=%v", ute.Type, ute.Value, ute.Field, ute.Offset)).SetInternal(err)
	} else if se, ok := err.(*json.SyntaxError); ok {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Syntax error: offset=%v, error=%v", se.Offset, se.Error())).SetInternal(err)
	}
	return err
}

func NewRouter(h *Handler) (*echo.Echo, error) {
	docs.SwaggerInfo.Host = h.Config.Address

	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.JSONSerializer = customJSONSerializer{}
	e.Validator = customValidator{
		validator: validator.New(),
	}
	e.IPExtractor = echo.ExtractIPFromXFFHeader(
		echo.TrustLoopback(false),   // e.g. ipv4 start with 127.
		echo.TrustLinkLocal(false),  // e.g. ipv4 start with 169.254
		echo.TrustPrivateNet(false), // e.g. ipv4 start with 10. or 192.168
	)

	templates, err := template.ParseFS(h.FileSystem, "web/templates/**/*.tmpl")
	if err != nil {
		return nil, fmt.Errorf("could not parse templates: %w", err)
	}
	e.Renderer = customRenderer{
		templates: templates,
	}

	setUpMiddleware(e, h)
	setUpRoutes(e, h)
	return e, nil
}

func setUpMiddleware(e *echo.Echo, h *Handler) {
	//Pre-router middlewares
	e.Pre(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: h.Config.AllowedOrigins,
	}))
	e.Pre(middleware.CSRF())
	if h.Config.RateLimitPerMinute > 0 {
		e.Pre(middleware.RateLimiterWithConfig(middleware.RateLimiterConfig{
			Store: middleware.NewRateLimiterMemoryStoreWithConfig(middleware.RateLimiterMemoryStoreConfig{
				Rate:      rate.Limit(h.Config.RateLimitPerMinute),
				ExpiresIn: time.Minute,
			})}))
	}
	e.Pre(middleware.Secure())
	e.Pre(middleware.StaticWithConfig(middleware.StaticConfig{
		Root:       "web/public",
		Filesystem: http.FS(h.FileSystem),
	}))
	e.Pre(middleware.TimeoutWithConfig(middleware.TimeoutConfig{
		Timeout: 10 * time.Second, Skipper: func(c echo.Context) bool {
			return strings.HasPrefix(c.Request().URL.Path, "/debug/pprof")
		},
	}))
	e.Pre(session.Middleware(sessions.NewCookieStore([]byte(h.Config.SessionSecret))))
	e.Pre(middleware.RequestIDWithConfig(middleware.RequestIDConfig{
		Generator: func() string {
			return id.New(id.Request)
		},
	}))
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
	e.Pre(middleware.RemoveTrailingSlash())

	//Post-router middlewares
	e.Use(middleware.GzipWithConfig(middleware.GzipConfig{
		Skipper: func(c echo.Context) bool {
			return !strings.Contains(c.Request().Header.Get("Accept-Encoding"), "gzip") || strings.HasPrefix(c.Path(), "/metrics")
		},
	}))
	e.Use(middleware.Decompress())
	e.Use(middleware.RecoverWithConfig(middleware.RecoverConfig{
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
	e.Use(echoprometheus.NewMiddleware("api"))
	pprof.Register(e)
}

func setUpRoutes(e *echo.Echo, h *Handler) {
	e.GET("/metrics", echoprometheus.NewHandler())
	e.GET("/swagger/*", echoSwagger.EchoWrapHandler(func(c *echoSwagger.Config) {
		c.SyntaxHighlight = true
	}))
	e.GET("/ping", h.GetPing)
	e.GET("/config", h.GetConfig)
	e.GET("/_", h.GetAdmin, h.RequiresAuth(RoleAdmin))
	e.GET("/", h.GetHome)

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
}
