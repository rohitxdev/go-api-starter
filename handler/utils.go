package handler

import (
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/rohitxdev/go-api-starter/repo"
)

// bindAndValidate binds path params, query params and the request body into provided type `i` and validates provided `i`. `i` must be a pointer. The default binder binds body based on Content-Type header. Validator must be registered using `Echo#Validator`.
func bindAndValidate(c echo.Context, i any) error {
	var err error
	if err = c.Bind(i); err != nil {
		return err
	}
	binder := echo.DefaultBinder{}
	if err = binder.BindHeaders(c, i); err != nil {
		return err
	}
	if err = c.Validate(i); err != nil {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, err)
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

type response struct {
	Message string `json:"message,omitempty"`
}

func createSession(c echo.Context, duration time.Duration, userId uint64) (*sessions.Session, error) {
	sess, err := session.Get("session", c)
	if err != nil {
		return nil, err
	}
	sess.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   int(duration.Seconds()),
		HttpOnly: true,
	}
	sess.Values["userId"] = userId
	if err := sess.Save(c.Request(), c.Response()); err != nil {
		return nil, err
	}
	return sess, nil
}

func getUser(c echo.Context) *repo.User {
	user, ok := c.Get("user").(*repo.User)
	if !ok {
		return nil
	}
	return user
}
