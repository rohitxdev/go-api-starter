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
		"appEnv":  h.Config.AppEnv,
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
		AppEnv: h.Config.AppEnv,
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
