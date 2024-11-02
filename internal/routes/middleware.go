package routes

import (
	"net/http"

	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
)

type role uint8

const (
	RoleUser role = iota
	RoleAdmin
)

var roleMap = map[string]role{
	"user":  RoleUser,
	"admin": RoleAdmin,
}

func RestrictTo(svc *Services, role role) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			sess, err := session.Get("session", c)
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, err)
			}
			userID, ok := sess.Values["userId"].(uint64)
			if !ok {
				return echo.ErrUnauthorized
			}
			user, err := svc.Repo.GetUserById(c.Request().Context(), userID)
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, err)
			}
			if (roleMap[user.Role] < role) || (user.AccountStatus != "active") {
				return echo.ErrForbidden
			}
			c.Set("user", user)
			return next(c)
		}
	}
}
