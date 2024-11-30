package handler

import (
	"fmt"
	"html/template"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/go-playground/validator"
	gojson "github.com/goccy/go-json"
	"github.com/labstack/echo-contrib/echoprometheus"
	"github.com/labstack/echo-contrib/pprof"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/oklog/ulid/v2"
	"github.com/rohitxdev/go-api-starter/assets"
	"github.com/rohitxdev/go-api-starter/docs"
	"github.com/rohitxdev/go-api-starter/repo"
	echoSwagger "github.com/swaggo/echo-swagger"
)

// Custom view renderer
type renderer struct {
	templates *template.Template
}

func (t renderer) Render(w io.Writer, name string, data any, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name+".tmpl", data)
}

// Custom request validator
type requestValidator struct {
	validator *validator.Validate
}

func (v requestValidator) Validate(i any) error {
	if err := v.validator.Struct(i); err != nil {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, err)
	}
	return nil
}

// Custom JSON serializer & deserializer
type jsonSerializer struct{}

func (s jsonSerializer) Serialize(c echo.Context, data any, indent string) error {
	enc := gojson.NewEncoder(c.Response())
	enc.SetIndent("", indent)
	return enc.Encode(data)
}

func (s jsonSerializer) Deserialize(c echo.Context, v any) error {
	dec := gojson.NewDecoder(c.Request().Body)
	err := dec.Decode(v)
	if ute, ok := err.(*gojson.UnmarshalTypeError); ok {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Unmarshal type error: expected=%v, got=%v, field=%v, offset=%v", ute.Type, ute.Value, ute.Field, ute.Offset)).SetInternal(err)
	} else if se, ok := err.(*gojson.SyntaxError); ok {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Syntax error: offset=%v, error=%v", se.Offset, se.Error())).SetInternal(err)
	}
	return err
}

func mountRoutes(e *echo.Echo, h *Handler) {
	e.GET("/metrics", echoprometheus.NewHandler())
	e.GET("/swagger/*", echoSwagger.EchoWrapHandler())
	e.GET("/config", h.GetConfig)
	e.GET("/me", h.GetMe)
	e.GET("/_", h.GetAdmin)
	e.GET("/", h.GetHome)

	auth := e.Group("/auth")
	{
		auth.POST("/sign-up", h.SignUp)
		auth.POST("/log-in", h.LogIn)
		auth.GET("/log-out", h.LogOut)
		auth.GET("/access-token", h.GetAccessToken)
		auth.PUT("/change-password", h.ChangePassword)
		auth.POST("/reset-password", h.SendResetPasswordEmail)
		auth.PUT("/reset-password", h.ResetPassword)
		auth.POST("/verify-account", h.SendAccountVerificationEmail)
		auth.PUT("/verify-account", h.VerifyAccount)
	}
}

func New(svc *Service) (*echo.Echo, error) {
	h := Handler{Service: svc}

	docs.SwaggerInfo.Host = net.JoinHostPort(h.Config.Host, h.Config.Port)

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
		return nil, fmt.Errorf("could not parse templates: %w", err)
	}
	e.Renderer = renderer{
		templates: pageTemplates,
	}

	e.HTTPErrorHandler = func(err error, c echo.Context) {
		defer func() {
			if err != nil {
				h.Logger.Error().
					Ctx(c.Request().Context()).
					Err(err).
					Str("id", c.Response().Header().Get(echo.HeaderXRequestID)).
					Msg("HTTP error response failure")
			}
		}()

		var res Response
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
			res.Message = "Something went wrong"
			err = c.JSON(http.StatusInternalServerError, res)
		}
	}

	//Pre-router middlewares
	if !h.Config.IsDev {
		e.Pre(middleware.CSRF())
		e.Pre(middleware.Secure())
	}

	e.Pre(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:                             h.Config.AllowedOrigins,
		AllowCredentials:                         true,
		UnsafeWildcardOriginWithAllowCredentials: h.Config.IsDev,
	}))

	e.Pre(middleware.StaticWithConfig(middleware.StaticConfig{
		Root:       "public",
		Filesystem: http.FS(assets.FS),
	}))

	// This middleware causes data races. See https://github.com/labstack/echo/issues/1761. But it's not a big deal.
	e.Pre(middleware.TimeoutWithConfig(middleware.TimeoutConfig{
		Timeout: 10 * time.Second, Skipper: func(c echo.Context) bool {
			return strings.HasPrefix(c.Path(), "/debug/pprof")
		},
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
			log := h.Logger.Info().
				Ctx(c.Request().Context()).
				Str("id", v.RequestID).
				Str("clientIp", v.RemoteIP).
				Str("protocol", v.Protocol).
				Str("uri", v.URI).
				Str("method", v.Method).
				Int64("durationMs", v.Latency.Milliseconds()).
				Int64("bytesOut", v.ResponseSize).
				Int("status", v.Status).
				Str("host", v.Host)

			if v.UserAgent != "" {
				log = log.Str("ua", v.UserAgent)
			}
			if v.Referer != "" {
				log = log.Str("referer", v.Referer)
			}
			if user, ok := c.Get("user").(*repo.User); ok && (user != nil) {
				log = log.Uint64("userId", user.ID)
			}

			if err, ok := v.Error.(*echo.HTTPError); ok {
				if err.Code == http.StatusInternalServerError {
					log = log.Err(err)
				}
			} else if v.Error != nil {
				// Due to a bug in echo, when the error is not an echo.HTTPError, even though the status code sent is 500, it's logged as 200 in this middleware. We need to manually set the status code in the log to 500.
				log = log.Err(v.Error).Int("status", http.StatusInternalServerError)
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
			h.Logger.Error().Ctx(c.Request().Context()).
				Err(err).
				Str("stack", string(stack)).
				Str("id", c.Response().Header().Get(echo.HeaderXRequestID)).
				Msg("HTTP handler panic")
			return nil
		}},
	))

	e.Use(echoprometheus.NewMiddleware("api"))

	pprof.Register(e)

	mountRoutes(e, &h)

	return e, nil
}
