package routes

import (
	"embed"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/rohitxdev/go-api-starter/internal/blobstore"
	"github.com/rohitxdev/go-api-starter/internal/config"
	"github.com/rohitxdev/go-api-starter/internal/email"
	"github.com/rohitxdev/go-api-starter/internal/kvstore"
	"github.com/rohitxdev/go-api-starter/internal/repo"
	"github.com/rs/zerolog"
)

type Services struct {
	BlobStore  *blobstore.Store
	Config     *config.Config
	EmbeddedFS *embed.FS
	Email      *email.Client
	KVStore    *kvstore.Store
	Logger     *zerolog.Logger
	Repo       *repo.Repo
}

// @Summary Home Page
// @Description Home page.
// @Router / [get]
// @Success 200 {html} string "home page"
func GetHome(svc *Services) echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.Render(http.StatusOK, "home.tmpl", nil)
	}
}

// @Summary Ping
// @Description Ping the server.
// @Router /ping [get]
// @Success 200 {object} response
func GetPing(svc *Services) echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.JSON(http.StatusOK, response{Message: "pong"})
	}
}

// @Summary Get config
// @Description Get client config.
// @Router /config [get]
// @Success 200 {object} map[string]any
func GetConfig(svc *Services) echo.HandlerFunc {
	clientConfig := map[string]any{
		"env":        svc.Config.Env,
		"appName":    svc.Config.AppName,
		"appVersion": svc.Config.AppVersion,
	}
	return func(c echo.Context) error {
		return c.JSON(http.StatusOK, clientConfig)
	}
}

// @Summary Admin route
// @Description Admin route.
// @Security ApiKeyAuth
// @Router /_ [get]
// @Success 200 {string} string "Admin page"
// @Failure 401 {string} string "invalid session"
func GetAdmin(svc *Services) echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.String(http.StatusOK, "Admin page")
	}
}

// @Summary Get user
// @Description Get user.
// @Security ApiKeyAuth
// @Router /me [get]
// @Success 200 {object} repo.User
// @Failure 401 {string} string "invalid session"
func GetMe(svc *Services) echo.HandlerFunc {
	return func(c echo.Context) error {
		user := getUser(c)
		if user == nil {
			return echo.NewHTTPError(http.StatusUnauthorized, "User is not logged in")
		}
		return c.JSON(http.StatusOK, user)
	}
}
