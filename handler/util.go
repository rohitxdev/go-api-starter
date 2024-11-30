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

func canonicalizeEmail(email string) string {
	email = strings.TrimSpace(email)
	email = strings.ToLower(email)
	parts := strings.Split(email, "@")
	username := parts[0]
	domain := parts[1]
	if strings.Contains(username, "+") {
		username = strings.Split(username, "+")[0]
	}
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
		return nil, echo.NewHTTPError(http.StatusUnauthorized, "User is not logged in")
	}
	userID, err := cryptoutil.VerifyJWT(accessTokenCookie.Value, h.Config.AccessTokenSecret)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusUnauthorized, "JWT verification failed")
	}
	user, err := h.Repo.GetUserById(c.Request().Context(), userID)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusNotFound, "User not found")
	}
	if user.AccountStatus != repo.AccountStatusActive {
		return nil, echo.NewHTTPError(http.StatusForbidden, "Account status is not ACTIVE")
	}
	if roles[role(user.Role)] < roles[role(r)] {
		return nil, echo.NewHTTPError(http.StatusForbidden, "Insufficient privileges")
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
		SameSite: http.SameSiteNoneMode,
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
		SameSite: http.SameSiteNoneMode,
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
		SameSite: http.SameSiteNoneMode,
		HttpOnly: true,
		Secure:   true,
	})
	c.SetCookie(&http.Cookie{
		Name:     "refreshToken",
		Value:    "",
		MaxAge:   -1,
		Path:     "/",
		SameSite: http.SameSiteNoneMode,
		HttpOnly: true,
		Secure:   true,
	})
}
