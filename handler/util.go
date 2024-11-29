package handler

import (
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/rohitxdev/go-api-starter/cryptoutil"
	"github.com/rohitxdev/go-api-starter/repo"
)

type Response struct {
	Message string `json:"message,omitempty"`
	Success bool   `json:"success"`
}

type ResponseWithPayload[T any] struct {
	Payload T      `json:"payload,omitempty"`
	Message string `json:"message,omitempty"`
	Success bool   `json:"success"`
}

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

type role string

const (
	RoleUser  role = repo.RoleUser
	RoleAdmin role = repo.RoleAdmin
)

var roles = map[role]uint8{
	RoleUser:  1,
	RoleAdmin: 2,
}

func (h Handler) checkAuth(c echo.Context, r role) (*repo.User, error) {
	accessTokenCookie, err := c.Cookie("accessToken")
	if err != nil {
		return nil, echo.ErrUnauthorized
	}
	userID, err := cryptoutil.VerifyJWT(accessTokenCookie.Value, h.Config.AccessTokenSecret)
	if err != nil {
		return nil, echo.ErrUnauthorized
	}
	user, err := h.Repo.GetUserById(c.Request().Context(), userID)
	if err != nil {
		return nil, echo.ErrUnauthorized
	}
	if user.AccountStatus != "active" {
		return nil, echo.ErrForbidden
	}
	if roles[role(user.Role)] < roles[role(r)] {
		return nil, echo.ErrForbidden
	}
	return user, nil
}

func setAccessTokenCookie(c echo.Context, expiresIn time.Duration, userID uint64, secret string) error {
	accessToken, err := cryptoutil.GenerateJWT(userID, expiresIn, secret)
	if err != nil {
		return err
	}
	cookie := http.Cookie{
		Name:     "accessToken",
		Value:    accessToken,
		MaxAge:   int(expiresIn.Seconds()),
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
	}
	c.SetCookie(&cookie)
	return nil
}

func setRefreshTokenCookie(c echo.Context, expiresIn time.Duration, userID uint64, secret string) error {
	refreshToken, err := cryptoutil.GenerateJWT(userID, expiresIn, secret)
	if err != nil {
		return err
	}
	cookie := http.Cookie{
		Name:     "refreshToken",
		Value:    refreshToken,
		MaxAge:   int(expiresIn.Seconds()),
		Path:     "/auth/access-token",
		HttpOnly: true,
		Secure:   true,
	}
	c.SetCookie(&cookie)
	return nil
}

func clearAuthCookies(c echo.Context) {
	c.SetCookie(&http.Cookie{
		Name:     "accessToken",
		Value:    "",
		MaxAge:   -1,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
	})
	c.SetCookie(&http.Cookie{
		Name:     "refreshToken",
		Value:    "",
		MaxAge:   -1,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
	})
}
