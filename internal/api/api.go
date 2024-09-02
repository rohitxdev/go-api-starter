package api

import (
	"embed"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/rohitxdev/go-api-template/pkg/config"
	"github.com/rohitxdev/go-api-template/pkg/email"
	"github.com/rohitxdev/go-api-template/pkg/repo"
	"github.com/rohitxdev/go-api-template/pkg/sqlite"
	"github.com/rohitxdev/go-api-template/pkg/storage"
)

type Handler struct {
	config   *config.Server
	kv       *sqlite.KV
	repo     *repo.Repo
	email    *email.Client
	fs       *storage.Client
	staticFS *embed.FS
}

func New(c *config.Server, kv *sqlite.KV, r *repo.Repo, email *email.Client, fs *storage.Client, staticFS *embed.FS) *Handler {
	return &Handler{
		config:   c,
		kv:       kv,
		repo:     r,
		email:    email,
		fs:       fs,
		staticFS: staticFS,
	}
}

// bindAndValidate binds path params, query params and the request body into provided type `i` and validates provided `i`. The default binder binds body based on Content-Type header. Validator must be registered using `Echo#Validator`.
func bindAndValidate(c echo.Context, i any) error {
	var err error
	if err = c.Bind(i); err != nil {
		_ = c.String(http.StatusInternalServerError, err.Error())
		return err
	}
	binder := echo.DefaultBinder{}
	if err = binder.BindHeaders(c, i); err != nil {
		_ = c.String(http.StatusInternalServerError, err.Error())
		return err
	}
	if err = c.Validate(i); err != nil {
		_ = c.String(http.StatusUnprocessableEntity, err.Error())
		return err
	}
	return err
}

var logOutCookie = &http.Cookie{
	Name:     "refresh-token",
	Secure:   true,
	HttpOnly: true,
	SameSite: http.SameSiteStrictMode,
	Path:     "/v1/auth/refresh-token",
	MaxAge:   0,
}

func createLogOutCookie() *http.Cookie {
	return logOutCookie
}

func createLogInCookie(refreshToken string, expiresIn time.Duration) *http.Cookie {
	return &http.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Path:     "/v1/auth/refresh-token",
		MaxAge:   int(expiresIn / time.Second),
	}
}
