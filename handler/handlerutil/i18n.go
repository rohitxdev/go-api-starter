package handlerutil

import (
	"github.com/labstack/echo/v4"
	"golang.org/x/text/language"
)

func Language(c echo.Context) string {
	lang, ok := c.Get("language").(string)
	if !ok {
		return language.English.String()
	}

	return lang
}
