package handler

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/rohitxdev/go-api/handler/handlerutil"
	"github.com/rohitxdev/go-api/handler/middleware"
	"github.com/rohitxdev/go-api/util"
)

const (
	CookieDeviceID          = "device_id"
	CookieDeviceIDSignature = "device_id_signature"
)

func (h *Handler) Home(c echo.Context) error {
	cfg := h.Config.Get()
	return c.Render(http.StatusOK, "home", echo.Map{
		"appName":    cfg.AppName,
		"appVersion": cfg.AppVersion,
	})
}

func verifyDeviceID(c echo.Context, secret string) bool {
	id := c.Request().Header.Get(middleware.HeaderXDeviceID)
	if id == "" {
		cookie, err := c.Cookie(CookieDeviceID)
		if err == nil {
			id = cookie.Value
		}
	}

	signature := c.Request().Header.Get(middleware.HeaderXDeviceIDSignature)
	if signature == "" {
		cookie, err := c.Cookie(CookieDeviceIDSignature)
		if err == nil {
			signature = cookie.Value
		}
	}

	return id != "" && signature != "" && util.VerifyHMAC(id, signature, secret)
}

func (h *Handler) Bootstrap(c echo.Context) error {
	type user struct {
		ID string `json:"string"`
	}
	type clientConfig struct {
		AppName    string `json:"app_name"`
		AppVersion string `json:"app_version"`
		BuildType  string `json:"build_type"`
	}
	var payload struct {
		DeviceID          string        `json:"device_id,omitzero"`
		DeviceIDSignature string        `json:"device_id_signature,omitzero"`
		User              *user         `json:"user,omitzero"`
		Config            *clientConfig `json:"config,omitzero"`
	}

	if !verifyDeviceID(c, h.Config.Get().DeviceIDSecret) {
		id, err := uuid.NewV7()
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError).SetInternal(fmt.Errorf("failed to generate UUID: %w", err))
		}

		deviceID := id.String()
		deviceIDSignature := util.SignHMAC(deviceID, h.Config.Get().DeviceIDSecret)

		if c.Request().Header.Get("X-Client-Type") == "mobile" {
			payload.DeviceID = deviceID
			payload.DeviceIDSignature = deviceIDSignature
		} else {
			c.SetCookie(&http.Cookie{
				Name:     CookieDeviceID,
				Value:    deviceID,
				Secure:   true,
				HttpOnly: true,
				SameSite: http.SameSiteNoneMode,
			})
			c.SetCookie(&http.Cookie{
				Name:     CookieDeviceIDSignature,
				Value:    deviceIDSignature,
				Secure:   true,
				HttpOnly: true,
				SameSite: http.SameSiteNoneMode,
			})
		}
	}

	if currentUser := handlerutil.CurrentUser(c, h.Repo); currentUser != nil {
		payload.User = &user{
			ID: currentUser.ID.String(),
		}
	}

	cfg := h.Config.Get()
	payload.Config = &clientConfig{
		AppName:    cfg.AppName,
		AppVersion: cfg.AppVersion,
		BuildType:  cfg.BuildType,
	}

	return c.JSON(http.StatusOK, APISuccessResponse{
		Data: payload,
	})
}
