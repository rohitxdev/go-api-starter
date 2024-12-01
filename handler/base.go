package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

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
