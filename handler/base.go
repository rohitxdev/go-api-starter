package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func (h *Handler) Home(c echo.Context) error {
	return c.Render(http.StatusOK, "home", echo.Map{
		"appName":    cfg.AppName,
		"appVersion": cfg.AppVersion,
	})
}

func (h *Handler) GetHealth(c echo.Context) error {
	return c.JSON(200, APIResponse{Success: true})
}

func (h *Handler) GetMe(c echo.Context) error {
	user := getCurrentUser(c, h.Repo)
	if user == nil {
		return c.JSON(http.StatusUnauthorized, APIResponse{
			Success: false,
			Message: "user not authenticated",
		})
	}

	return c.JSON(http.StatusOK, APIResponseWithPayload{
		Success: true,
		Message: "fetched user successfully",
		Payload: user,
	})
}

type clientConfig struct {
	AppName    string `json:"app_name"`
	AppVersion string `json:"app_version"`
	BuildType  string `json:"build_type"`
	CSRFToken  string `json:"csrf_token"`
}

func (h *Handler) GetConfig(c echo.Context) error {
	cfg := clientConfig{
		AppName:    cfg.AppName,
		AppVersion: cfg.AppVersion,
		BuildType:  cfg.BuildType,
	}
	if csrfToken, ok := c.Get("csrf").(string); ok {
		cfg.CSRFToken = csrfToken
	}
	return c.JSON(http.StatusOK, APIResponseWithPayload{
		Success: true,
		Message: "fetched client config successfully",
		Payload: cfg,
	})
}
