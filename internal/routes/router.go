package routes

import (
	"fmt"
	"html/template"
	"io"
	"net"
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
	"github.com/oklog/ulid/v2"
	"github.com/rohitxdev/go-api-starter/docs"
	"github.com/rohitxdev/go-api-starter/internal/repo"
	echoSwagger "github.com/swaggo/echo-swagger"
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

func NewRouter(svc *Services) (*echo.Echo, error) {
	docs.SwaggerInfo.Host = net.JoinHostPort(svc.Config.Host, svc.Config.Port)

	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.Debug = svc.Config.IsDev
	e.JSONSerializer = customJSONSerializer{}
	e.Validator = customValidator{
		validator: validator.New(),
	}
	e.IPExtractor = echo.ExtractIPFromXFFHeader(
		echo.TrustLoopback(false),   // e.g. ipv4 start with 127.
		echo.TrustLinkLocal(false),  // e.g. ipv4 start with 169.254
		echo.TrustPrivateNet(false), // e.g. ipv4 start with 10. or 192.168
	)

	pageTemplates, err := template.ParseFS(svc.FileSystem, "templates/pages/*.tmpl")
	if err != nil {
		return nil, fmt.Errorf("could not parse templates: %w", err)
	}
	e.Renderer = customRenderer{
		templates: pageTemplates,
	}

	//Pre-router middlewares
	if !svc.Config.IsDev {
		e.Pre(middleware.CSRF())
	}

	e.Pre(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:                             svc.Config.AllowedOrigins,
		AllowCredentials:                         true,
		UnsafeWildcardOriginWithAllowCredentials: svc.Config.IsDev,
	}))

	e.Pre(middleware.Secure())

	e.Pre(middleware.StaticWithConfig(middleware.StaticConfig{
		Root:       "public",
		Filesystem: http.FS(svc.FileSystem),
	}))

	e.Pre(middleware.TimeoutWithConfig(middleware.TimeoutConfig{
		Timeout: 10 * time.Second, Skipper: func(c echo.Context) bool {
			return strings.HasPrefix(c.Request().URL.Path, "/debug/pprof")
		},
	}))

	e.Pre(session.Middleware(sessions.NewCookieStore([]byte(svc.Config.SessionSecret))))

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
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			log := svc.Logger.Info().
				Ctx(c.Request().Context()).
				Str("id", v.RequestID).
				Str("ip", v.RemoteIP).
				Str("protocol", v.Protocol).
				Str("uri", v.URI).
				Str("method", v.Method).
				Int64("durationMs", v.Latency.Milliseconds()).
				Int64("resBytes", v.ResponseSize).
				Int("status", v.Status).
				Err(v.Error)

			if user, ok := c.Get("user").(*repo.User); ok && (user != nil) {
				log = log.Uint64("userId", user.Id)
			}
			if v.UserAgent != "" {
				log = log.Str("ua", v.UserAgent)
			}
			if v.Referer != "" {
				log = log.Str("referer", v.Referer)
			}

			log.Msg("HTTP request")

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
			svc.Logger.Error().Ctx(c.Request().Context()).
				Any("error", err).
				Str("stack", string(stack)).
				Str("method", c.Request().Method).
				Str("path", c.Path()).
				Str("ip", c.RealIP()).
				Msg("HTTP handler panicked")
			return nil
		}},
	))

	e.Use(echoprometheus.NewMiddleware("api"))

	pprof.Register(e)

	setUpRoutes(e, svc)

	return e, nil
}

func setUpRoutes(e *echo.Echo, svc *Services) {
	e.GET("/metrics", echoprometheus.NewHandler())
	e.GET("/swagger/*", echoSwagger.EchoWrapHandler(func(c *echoSwagger.Config) {
		c.SyntaxHighlight = true
	}))
	e.GET("/ping", GetPing(svc))
	e.GET("/config", GetConfig(svc))
	e.GET("/me", GetMe(svc), RestrictTo(svc, RoleUser))
	e.GET("/_", GetAdmin(svc), RestrictTo(svc, RoleAdmin))
	e.GET("/", GetHome(svc))

	auth := e.Group("/auth")
	{
		logIn := auth.Group("/log-in")
		{
			logIn.GET("", ValidateLogInToken(svc))
			logIn.POST("", SendLoginEmail(svc))
		}
		auth.GET("/log-out", LogOut(svc))
	}
}
