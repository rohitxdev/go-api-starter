package middleware

import (
	"net/http"
	"runtime/debug"

	"github.com/labstack/echo/v4"
)

const (
	CtxKeyPanicReason = "panic_reason"
	CtxKeyPanicStack  = "panic_stack"
)

func RecoverPanic() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) (err error) {
			defer func() {
				if reason := recover(); reason != nil {
					c.Set(CtxKeyPanicReason, reason)
					c.Set(CtxKeyPanicStack, string(debug.Stack()))
					err = echo.NewHTTPError(http.StatusInternalServerError)
				}
			}()

			err = next(c)
			return
		}
	}
}
