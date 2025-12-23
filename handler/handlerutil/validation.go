package handlerutil

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func BindAndValidate(c echo.Context, i any) error {
	if err := c.Bind(i); err != nil {
		return err
	}

	binder := echo.DefaultBinder{}
	if err := binder.BindHeaders(c, i); err != nil {
		return err
	}

	if err := c.Validate(i); err != nil {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, err)
	}

	return nil
}
