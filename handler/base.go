package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/rohitxdev/go-api/handler/handlerutil"
)

func (h *Handler) Home(c echo.Context) error {
	return c.Render(http.StatusOK, "home", echo.Map{
		"appName":    h.Config.AppName,
		"appVersion": h.Config.AppVersion,
	})
}

func (h *Handler) GetMe(c echo.Context) error {
	user := handlerutil.CurrentUser(c, h.Repo)
	if user == nil {
		return c.JSON(http.StatusUnauthorized, APIErrorResponse{
			Error: "user not authenticated",
		})
	}

	return c.JSON(http.StatusOK, APISuccessResponse{
		Data: user,
	})
}

func (h *Handler) GetConfig(c echo.Context) error {
	type clientConfig struct {
		AppName    string `json:"app_name"`
		AppVersion string `json:"app_version"`
		BuildType  string `json:"build_type"`
	}

	return c.JSON(http.StatusOK, APISuccessResponse{
		Data: clientConfig{
			AppName:    h.Config.AppName,
			AppVersion: h.Config.AppVersion,
			BuildType:  h.Config.BuildType,
		},
	})
}
