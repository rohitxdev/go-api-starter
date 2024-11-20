package handler

import (
	"net/http"
	"slices"

	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
)

type role string

const (
	RoleUser  role = "user"
	RoleAdmin role = "admin"
)

type permission string

const (
	PermReadAdmin permission = "read:admin"
	PermReadMe    permission = "read:me"
)

var permissions = map[role][]permission{
	RoleUser:  {PermReadMe},
	RoleAdmin: {PermReadAdmin, PermReadMe},
}

func (h *Handler) require(permission permission) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			sess, err := session.Get("session", c)
			if err != nil {
				return err
			}
			userID, ok := sess.Values["userId"].(uint64)
			if !ok {
				return echo.ErrUnauthorized
			}
			user, err := h.Repo.GetUserById(c.Request().Context(), userID)
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, err)
			}
			if (!slices.Contains(permissions[role(user.Role)], permission)) || (user.AccountStatus != "active") {
				return echo.ErrForbidden
			}
			c.Set("user", user)
			return next(c)
		}
	}
}
