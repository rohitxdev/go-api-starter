package routes

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/rohitxdev/go-api-starter/internal/auth"
	"github.com/rohitxdev/go-api-starter/internal/email"
	"github.com/rohitxdev/go-api-starter/internal/kv"
	"github.com/rohitxdev/go-api-starter/internal/repo"
)

const (
	logInTokenExpiresIn = time.Minute * 10 // 10 minutes
)

func (h *Handler) LogOut(c echo.Context) error {
	sess, err := session.Get("session", c)
	if err != nil {
		return err
	}
	if sess.IsNew {
		return echo.NewHTTPError(http.StatusBadRequest, "User is not logged in")
	}
	sess.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	}
	if err := sess.Save(c.Request(), c.Response()); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, response{Message: "Logged out successfully"})
}

type logInRequest struct {
	Token string `query:"token" validate:"required"`
}

type logInResponse struct {
	response
	UserId string `json:"userId"`
}

func (h *Handler) ValidateLogInToken(c echo.Context) error {
	req := new(logInRequest)
	if err := bindAndValidate(c, req); err != nil {
		return err
	}
	userId, err := auth.ValidateLoginToken(req.Token, h.Config.JwtSecret)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "Invalid token")
	}
	tokenKey := getLogInTokenKey(userId)
	token, err := h.KVStore.Get(c.Request().Context(), tokenKey)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "Invalid token")
	}
	if token != req.Token {
		return echo.NewHTTPError(http.StatusUnauthorized, "Invalid token")
	}

	user, err := h.Repo.GetUserById(c.Request().Context(), userId)
	if err != nil {
		return err
	}
	if !user.IsVerified {
		if err = h.Repo.SetIsVerified(c.Request().Context(), user.Id, true); err != nil {
			return err
		}
	}
	if err = h.KVStore.Delete(c.Request().Context(), tokenKey); err != nil {
		return err
	}
	if _, err = createSession(c, userId); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, logInResponse{
		response: response{Message: "Logged in successfully"},
		UserId:   userId,
	})
}

type sendLoginEmailRequest struct {
	Email string `form:"email" json:"email" validate:"required,email"`
}

func (h *Handler) SendLoginEmail(c echo.Context) error {
	req := new(sendLoginEmailRequest)
	if err := bindAndValidate(c, req); err != nil {
		return err
	}
	host := c.Request().Host
	if host == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Host header is empty")
	}

	userEmail := sanitizeEmail(req.Email)
	var userId string
	user, err := h.Repo.GetUserByEmail(c.Request().Context(), userEmail)

	if err != nil {
		if !errors.Is(err, repo.ErrUserNotFound) {
			return fmt.Errorf("Failed to get user: %w", err)
		}
		userId, err = h.Repo.CreateUser(c.Request().Context(), userEmail)
		if err != nil {
			return fmt.Errorf("Failed to create user: %w", err)
		}
	} else {
		userId = user.Id
	}

	token, err := auth.GenerateLoginToken(userId, h.Config.JwtSecret)
	if err != nil {
		return fmt.Errorf("Failed to generate login token: %w", err)
	}

	emailHeaders := &email.Headers{
		Subject:     "Log In to Your Account",
		ToAddresses: []string{req.Email},
		FromAddress: h.Config.SenderEmail,
		FromName:    "The App",
	}
	protocol := "http"
	if c.IsTLS() {
		protocol = "https"
	}
	emailData := map[string]any{
		"loginURL":     fmt.Sprintf("%s://%s%s?token=%s", protocol, host, c.Path(), token),
		"validMinutes": uint(logInTokenExpiresIn.Minutes()),
	}
	if err = h.Email.SendHtml(emailHeaders, "login.tmpl", emailData); err != nil {
		return fmt.Errorf("Failed to send email: %w", err)
	}
	if err = h.KVStore.Set(c.Request().Context(), getLogInTokenKey(userId), token, kv.WithExpiry(logInTokenExpiresIn)); err != nil {
		return fmt.Errorf("Failed to set token: %w", err)
	}
	return c.JSON(http.StatusOK, response{Message: "Login link sent to " + req.Email})
}

func getLogInTokenKey(userId string) string {
	return "login.token." + userId
}
