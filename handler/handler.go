package handler

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"text/template"
	"time"

	"github.com/go-redis/redis_rate/v10"
	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/echoprometheus"
	"github.com/labstack/echo-contrib/pprof"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
	"github.com/redis/go-redis/v9"
	"github.com/rohitxdev/go-api/assets"
	"github.com/rohitxdev/go-api/database/repository"
	"github.com/rohitxdev/go-api/deps/blobstore"
	"github.com/rohitxdev/go-api/deps/cache"
	"github.com/rohitxdev/go-api/deps/config"
	"github.com/rohitxdev/go-api/deps/email"
	"github.com/rohitxdev/go-api/handler/middleware"
	"github.com/rohitxdev/go-api/util"
)

const (
	HeaderXClientID = "X-Client-ID"
	HeaderXTraceID  = "X-Trace-ID"
)

type Dependencies struct {
	BlobStore   *blobstore.BlobStore
	Config      *config.Store
	Cache       *cache.Cache[string]
	Email       *email.Client
	Logger      *slog.Logger
	Redis       *redis.Client
	Repo        *repository.Queries
	RateLimiter *redis_rate.Limiter
}

type Handler struct {
	*Dependencies
}

func registerV1Routes(e *echo.Echo, h *Handler) {
	v1 := e.Group("/v1")
	{
		v1.GET("/bootstrap", h.Bootstrap)
		v1.GET("/metrics", echoprometheus.NewHandler())
		v1.GET("/", func(c echo.Context) error {
			return c.Redirect(http.StatusTemporaryRedirect, "/views/home")
		})

		views := v1.Group("/views")
		{
			views.GET("/home", h.Home)
		}

		auth := v1.Group("/auth")
		{
			signIn := auth.Group("/sign-in")
			{
				signIn.POST("/send-code", h.SendSignInVerificationCode)
				signIn.POST("/verify-code", h.VerifySignIn, middleware.RateLimit(h.RateLimiter, redis_rate.PerMinute(2)))
			}
			auth.POST("/sign-out", h.SignOut)
		}
	}
}

type countingReadCloser struct {
	rc io.ReadCloser
	n  int64
}

func (c *countingReadCloser) Read(buf []byte) (int, error) {
	n, err := c.rc.Read(buf)
	c.n += int64(n)
	return n, err
}

func (c *countingReadCloser) Close() error {
	return c.rc.Close()
}

func handleHTTPError(logger *slog.Logger) echo.HTTPErrorHandler {
	return func(err error, c echo.Context) {
		var (
			traceID     string
			panicReason any
			panicStack  string
			status      = http.StatusInternalServerError
			msg         = http.StatusText(status)
		)

		if v, ok := c.Get(middleware.CtxKeyTraceID).(string); ok {
			traceID = v
		}
		if v := c.Get(middleware.CtxKeyPanicReason); v != nil {
			panicReason = v
		}
		if v, ok := c.Get(middleware.CtxKeyPanicStack).(string); ok {
			panicStack = v
		}

		attrs := []any{slog.String("trace_id", traceID)}

		if panicReason != nil && panicStack != "" {
			msg = "panic recovered"
			attrs = append(attrs,
				slog.Any("error", panicReason),
				slog.String("call_stack", panicStack),
			)
		} else if httpErr, ok := err.(*echo.HTTPError); ok {
			switch httpErrMsg := httpErr.Message.(type) {
			case string:
				msg = httpErrMsg
			case error:
				msg = httpErrMsg.Error()
			default:
				msg = httpErr.Error()
			}

			if httpErr.Internal != nil {
				attrs = append(attrs, slog.String("error", httpErr.Internal.Error()))
			}

			status = httpErr.Code
		} else {
			status = http.StatusInternalServerError
			attrs = append(attrs, slog.String("error", err.Error()))
		}

		if status == http.StatusInternalServerError {
			msg = echo.ErrInternalServerError.Message.(string)
			logger.Error(msg, attrs...)
		}

		if err := c.JSON(status, APIErrorResponse{Error: msg}); err != nil {
			logger.Error("failed to send response",
				slog.String("trace_id", traceID),
				slog.String("error", err.Error()),
			)
		}
	}
}

func New(deps *Dependencies) (*echo.Echo, error) {
	h := Handler{Dependencies: deps}
	h.RateLimiter = redis_rate.NewLimiter(h.Redis)

	cfg := h.Config.Get()

	e := echo.New()

	pageTemplates, err := template.ParseFS(assets.FS, "templates/pages/*.tmpl")
	if err != nil {
		return nil, fmt.Errorf("failed to parse templates: %w", err)
	}

	e.Renderer = viewRenderer{
		templates: pageTemplates,
	}

	e.JSONSerializer = jsonSerializer{}
	e.Validator = requestValidator{
		validator: util.Validate,
	}
	e.IPExtractor = echo.ExtractIPFromXFFHeader(
		echo.TrustLoopback(false),   // e.g. ipv4 start with 127.
		echo.TrustLinkLocal(false),  // e.g. ipv4 start with 169.254
		echo.TrustPrivateNet(false), // e.g. ipv4 start with 10. or 192.168
	)
	e.HTTPErrorHandler = handleHTTPError(h.Logger)

	e.Pre(middleware.NormalizePath())

	e.Use(
		// logging
		middleware.LogRequest(h.Logger),
		// safety net
		middleware.RecoverPanic(),
		// identity
		middleware.ResolveIdentityKey(h.Redis, h.Config),
		// security
		middleware.RateLimit(h.RateLimiter, redis_rate.PerMinute(100)),
		echomiddleware.Secure(),
		echomiddleware.CORSWithConfig(echomiddleware.CORSConfig{
			AllowOrigins:     cfg.AllowedOrigins,
			AllowCredentials: true,
			MaxAge:           60,
		}),
		// request/response processing
		echomiddleware.Decompress(),
		echomiddleware.BodyLimit("4MB"),
		echomiddleware.Gzip(),
		// infra
		echomiddleware.TimeoutWithConfig(echomiddleware.TimeoutConfig{
			Timeout: time.Second * 10,
			Skipper: func(c echo.Context) bool {
				return strings.HasPrefix(c.Path(), "/debug/pprof")
			},
		}),
		// i18n
		middleware.ResolveLanguage(),
		// sessions
		session.Middleware(sessions.NewCookieStore([]byte(cfg.SessionSecret))),
		// metrics
		echoprometheus.NewMiddleware("api"),
		// static files
		echomiddleware.StaticWithConfig(echomiddleware.StaticConfig{
			Filesystem: http.FS(assets.FS),
			Root:       "/public",
		}),
	)

	pprof.Register(e)

	registerV1Routes(e, &h)

	return e, nil
}
