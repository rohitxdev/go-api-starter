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
func (h *Handler) GetHome(c echo.Context) error {
	return c.Render(http.StatusOK, "home", echo.Map{
		"appName":    h.Config.AppName,
		"appVersion": h.Config.AppVersion,
	})
}

type ClientConfig struct {
	Env        string `json:"env"`
	AppName    string `json:"appName"`
	AppVersion string `json:"appVersion"`
}

// @Summary Get config
// @Description Get client config.
// @Router /config [get]
// @Success 200 {object} ResponseWithPayload[ClientConfig]
func (h *Handler) GetConfig(c echo.Context) error {
	return c.JSON(http.StatusOK, ResponseWithPayload[ClientConfig]{
		Message: "Fetched config successfully",
		Success: true,
		Payload: ClientConfig{
			Env:        h.Config.Env,
			AppName:    h.Config.AppName,
			AppVersion: h.Config.AppVersion,
		},
	})
}

// @Summary Admin page
// @Description Admin page.
// @Security ApiKeyAuth
// @Router /_ [get]
// @Success 200 {string} string "Admin page"
// @Failure 401 {string} string "invalid session"
func (h *Handler) GetAdmin(c echo.Context) error {
	_, err := h.checkAuth(c, RoleAdmin)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, Response{
		Message: "You have admin privileges",
		Success: true,
	})
}

// @Summary Get user
// @Description Get user.
// @Security ApiKeyAuth
// @Router /me [get]
// @Success 200 {object} ResponseWithPayload[repo.PublicUser]
// @Failure 401 {string} string "invalid session"
func (h *Handler) GetMe(c echo.Context) error {
	user, err := h.checkAuth(c, RoleUser)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, ResponseWithPayload[repo.PublicUser]{
		Message: "Fetched current user successfully",
		Success: true,
		Payload: repo.PublicUser{
			ID:          user.ID,
			Role:        user.Role,
			Email:       user.Email,
			Username:    user.Username,
			ImageURL:    user.ImageURL,
			Gender:      user.Gender,
			DateOfBirth: user.DateOfBirth,
			CreatedAt:   user.CreatedAt,
			UpdatedAt:   user.UpdatedAt,
		},
	})
}
