package handler

import (
	"net/http"

	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
)

type role uint8

const (
	RoleUser role = iota + 1
	RoleAdmin
)

var roleMap = map[string]role{
	"user":  RoleUser,
	"admin": RoleAdmin,
}

func (h *handler) protected(role role) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			sess, err := session.Get("session", c)
			if err != nil {
				return c.String(http.StatusUnauthorized, err.Error())
			}
			userId, ok := sess.Values["user_id"].(string)
			if !ok {
				return c.String(http.StatusUnauthorized, "invalid session")
			}
			user, err := h.repo.GetUserById(c.Request().Context(), userId)
			if err != nil {
				return c.String(http.StatusUnauthorized, err.Error())
			}
			if roleMap[user.Role] < role {
				return c.String(http.StatusForbidden, "forbidden")
			}
			c.Set("user", user)
			return next(c)
		}
	}
}
