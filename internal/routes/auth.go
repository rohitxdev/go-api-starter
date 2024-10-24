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
	sessionMaxAge = 86400 * 30 // 30 days
)

var (
	ErrUserNotLoggedIn = errors.New("user is not logged in")
)

func createSession(c echo.Context, userId string) (*sessions.Session, error) {
	sess, err := session.Get("session", c)
	if err != nil {
		return nil, err
	}
	sess.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   sessionMaxAge,
		HttpOnly: true,
	}
	sess.Values["user_id"] = userId
	if err := sess.Save(c.Request(), c.Response()); err != nil {
		return nil, err
	}
	return sess, nil
}

func (h *Handler) LogOut(c echo.Context) error {
	sess, err := session.Get("session", c)
	if err != nil {
		return c.String(http.StatusBadRequest, ErrUserNotLoggedIn.Error())
	}
	sess.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	}
	if err := sess.Save(c.Request(), c.Response()); err != nil {
		return err
	}
	return c.String(http.StatusOK, "Logged out")
}

type logInRequest struct {
	Token string `query:"token"`
}

type logInResponse struct {
	response
	UserId string `json:"userId"`
}

func (h *Handler) LogIn(c echo.Context) error {
	req := new(logInRequest)
	if err := bindAndValidate(c, req); err != nil {
		return err
	}
	userId, err := auth.ValidateLoginToken(req.Token, h.Config.JwtSecret)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, response{Message: "Invalid token"})
	}
	tokenKey := getTokenKey(userId)
	token, err := h.KVStore.Get(c.Request().Context(), tokenKey)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, response{Message: "Invalid token"})
	}
	if token != req.Token {
		return c.JSON(http.StatusUnauthorized, response{Message: "Invalid token"})
	}

	user, err := h.Repo.GetUserById(c.Request().Context(), userId)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, response{Message: "Internal server error"})
	}
	if !user.IsVerified {
		if err = h.Repo.SetIsVerified(c.Request().Context(), user.Id, true); err != nil {
			return c.JSON(http.StatusInternalServerError, response{Message: "Internal server error"})
		}
	}
	if err = h.KVStore.Delete(c.Request().Context(), tokenKey); err != nil {
		return c.JSON(http.StatusInternalServerError, response{Message: "Internal server error"})
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
		return c.JSON(http.StatusBadRequest, response{Message: "Host header is empty"})
	}

	userEmail := sanitizeEmail(req.Email)
	var userId string
	user, err := h.Repo.GetUserByEmail(c.Request().Context(), userEmail)

	if err != nil {
		if !errors.Is(err, repo.ErrUserNotFound) {
			return c.JSON(http.StatusInternalServerError, response{Message: "Internal server error"})
		}
		userId, err = h.Repo.CreateUser(c.Request().Context(), userEmail)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, response{Message: "Failed to create user"})
		}
	} else {
		userId = user.Id
	}

	token, err := auth.GenerateLoginToken(userId, h.Config.JwtSecret)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, response{Message: "Failed to generate login token"})
	}

	emailHeaders := &email.Headers{
		Subject:     "Log In to Your Account",
		ToAddresses: []string{req.Email},
		FromAddress: h.Config.SenderEmail,
		FromName:    "The App",
	}
	emailData := map[string]any{
		"loginURL":     fmt.Sprintf("http://%s/v1/auth/log-in?token=%s", host, token),
		"validMinutes": "5",
	}
	if err = h.Email.SendHtml(emailHeaders, "login.tmpl", emailData); err != nil {
		return c.JSON(http.StatusInternalServerError, response{Message: "Failed to send email"})
	}
	if err = h.KVStore.Set(c.Request().Context(), getTokenKey(userId), token, kv.WithExpiry(time.Minute*5)); err != nil {
		return c.JSON(http.StatusInternalServerError, response{Message: "Failed to set token"})
	}
	return c.String(http.StatusOK, "Login link sent to "+req.Email)
}

func getTokenKey(userId string) string {
	return "login.token." + userId
}
