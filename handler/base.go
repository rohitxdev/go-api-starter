package handler

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/rohitxdev/go-api-starter/blobstore"
	"github.com/rohitxdev/go-api-starter/config"
	"github.com/rohitxdev/go-api-starter/email"
	"github.com/rohitxdev/go-api-starter/kvstore"
	"github.com/rohitxdev/go-api-starter/repo"
	"github.com/rs/zerolog"
)

type Service struct {
	BlobStore *blobstore.Store
	Config    *config.Config
	Email     *email.Client
	KVStore   *kvstore.Store
	Logger    *zerolog.Logger
	Repo      *repo.Repo
}

func (s *Service) Close() error {
	if err := s.KVStore.Close(); err != nil {
		return fmt.Errorf("Failed to close KV store: %w", err)
	}
	if err := s.Repo.Close(); err != nil {
		return fmt.Errorf("Failed to close repo: %w", err)
	}
	return nil
}

type Handler struct {
	*Service
}

// @Summary Home Page
// @Description Home page.
// @Router / [get]
// @Success 200 {html} string "home page"
func (h *Handler) getHome(c echo.Context) error {
	return c.Render(http.StatusOK, "home.tmpl", nil)
}

// @Summary Get config
// @Description Get client config.
// @Router /config [get]
// @Success 200 {object} map[string]any
func (h *Handler) getConfig(c echo.Context) error {
	cfg := h.Config
	clientConfig := map[string]any{
		"env":        cfg.Env,
		"appName":    cfg.AppName,
		"appVersion": cfg.AppVersion,
	}
	return c.JSON(http.StatusOK, clientConfig)
}

// @Summary Admin route
// @Description Admin route.
// @Security ApiKeyAuth
// @Router /_ [get]
// @Success 200 {string} string "Admin page"
// @Failure 401 {string} string "invalid session"
func (h *Handler) getAdmin(c echo.Context) error {
	_, err := h.checkAuth(c, RoleAdmin)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, response{Message: "You're an admin."})
}

// @Summary Get user
// @Description Get user.
// @Security ApiKeyAuth
// @Router /me [get]
// @Success 200 {object} repo.User
// @Failure 401 {string} string "invalid session"
func (h *Handler) getMe(c echo.Context) error {
	user, err := h.checkAuth(c, RoleUser)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, user)
}
