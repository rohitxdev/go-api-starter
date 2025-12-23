package middleware

import (
	"github.com/labstack/echo/v4"
	"golang.org/x/text/language"
)

const (
	HeaderAcceptLanguage = "Accept-Language"
)

// First language is the fallback value in case of no matches
var supportedLanguages = []language.Tag{
	language.English,
	language.Spanish,
	language.German,
	language.French,
	language.Italian,
}

var matcher = language.NewMatcher(supportedLanguages)

func ResolveLanguage() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			acceptLanguage := c.Request().Header.Get(HeaderAcceptLanguage)
			tags, _, _ := language.ParseAcceptLanguage(acceptLanguage)
			tag, _, _ := matcher.Match(tags...)
			base, _ := tag.Base()

			c.Set("language", base.String())

			return next(c)
		}
	}
}
