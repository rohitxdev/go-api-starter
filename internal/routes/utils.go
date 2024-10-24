package routes

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

// bindAndValidate binds path params, query params and the request body into provided type `i` and validates provided `i`. The default binder binds body based on Content-Type header. Validator must be registered using `Echo#Validator`.
func bindAndValidate(c echo.Context, i any) error {
	var err error
	if err = c.Bind(i); err != nil {
		_ = c.JSON(http.StatusInternalServerError, response{
			Message: err.Error(),
		})
		return err
	}
	binder := echo.DefaultBinder{}
	if err = binder.BindHeaders(c, i); err != nil {
		_ = c.JSON(http.StatusInternalServerError, response{
			Message: err.Error(),
		})
		return err
	}
	if err = c.Validate(i); err != nil {
		_ = c.JSON(http.StatusUnprocessableEntity, response{
			Message: err.Error(),
		})
		return err
	}
	return err
}

func sanitizeEmail(email string) string {
	emailParts := strings.Split(email, "@")
	username := emailParts[0]
	domain := emailParts[1]
	if strings.Contains(username, "+") {
		username = strings.Split(username, "+")[0]
	}
	username = strings.ReplaceAll(username, "-", "")
	username = strings.ReplaceAll(username, ".", "")
	return username + "@" + domain
}

func accepts(c echo.Context) string {
	acceptedTypes := strings.Split(c.Request().Header.Get("Accept"), ",")
	return acceptedTypes[0]
}

type response struct {
	Message string `json:"message"`
}
