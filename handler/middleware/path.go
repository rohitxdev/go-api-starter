package middleware

import (
	"strings"

	"github.com/labstack/echo/v4"
)

func NormalizePath() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			req := c.Request()
			path := req.URL.Path

			if path != "/" && strings.HasSuffix(path, "/") {
				path = strings.TrimRight(path, "/")
				if path == "" {
					path = "/"
				}
				req.URL.Path = path
				req.RequestURI = req.URL.RequestURI()
			}

			return next(c)
		}
	}
}
