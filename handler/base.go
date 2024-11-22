package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

type Handler struct {
	*Services
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
	return c.JSON(http.StatusOK, response{Message: "You're an admin."})
}

// @Summary Get user
// @Description Get user.
// @Security ApiKeyAuth
// @Router /me [get]
// @Success 200 {object} repo.User
// @Failure 401 {string} string "invalid session"
func (h *Handler) getMe(c echo.Context) error {
	user := getUser(c)
	if user == nil {
		return c.JSON(http.StatusUnauthorized, response{Message: "User is not logged in."})
	}
	return c.JSON(http.StatusOK, user)
}
