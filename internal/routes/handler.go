package routes

import (
	"embed"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/rohitxdev/go-api-starter/internal/blobstore"
	"github.com/rohitxdev/go-api-starter/internal/config"
	"github.com/rohitxdev/go-api-starter/internal/email"
	"github.com/rohitxdev/go-api-starter/internal/kv"
	"github.com/rohitxdev/go-api-starter/internal/repo"
)

type Dependencies struct {
	Config     *config.Server
	FileSystem *embed.FS
	Repo       *repo.Repo
	Email      *email.Client
	BlobStore  *blobstore.Store
	KVStore    *kv.Store
}

type Handler struct {
	*Dependencies
}

func NewHandler(deps *Dependencies) *Handler {
	return &Handler{Dependencies: deps}
}

// @Summary Home Page
// @Description Home page.
// @Router / [get]
// @Success 200 {html} string "home page"
func (h *Handler) GetHome(c echo.Context) error {
	data := echo.Map{
		"buildId": config.BuildId,
		"env":     h.Config.Env,
	}
	switch accepts(c) {
	case "text/html":
		return c.Render(http.StatusOK, "home.tmpl", data)
	default:
		return c.JSON(http.StatusOK, data)
	}
}

// @Summary Ping
// @Description Ping the server.
// @Router /ping [get]
// @Success 200 {string} string "pong"
func (h *Handler) GetPing(c echo.Context) error {
	return c.String(http.StatusOK, "pong")
}

// @Summary Get config
// @Description Get client config.
// @Router /config [get]
// @Success 200 {object} config.Client
func (h *Handler) GetConfig(c echo.Context) error {
	clientConfig := config.Client{
		Env: h.Config.Env,
	}
	return c.JSON(http.StatusOK, clientConfig)
}

// @Summary Admin route
// @Description Admin route.
// @Security ApiKeyAuth
// @Router /_ [get]
// @Success 200 {string} string "Admin page"
// @Failure 401 {string} string "invalid session"
func (h *Handler) GetAdmin(c echo.Context) error {
	return c.String(http.StatusOK, "Admin page")
}

// @Summary Get user
// @Description Get user.
// @Security ApiKeyAuth
// @Router /me [get]
// @Success 200 {object} repo.User
// @Failure 401 {string} string "invalid session"
func (h *Handler) GetMe(c echo.Context) error {
	user := getUser(c)
	if user == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "User is not logged in")
	}
	return c.JSON(http.StatusOK, user)
}
