package handler

import (
	"errors"
	"fmt"
	"net/http"
	"slices"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/rohitxdev/go-api-starter/auth"
	"github.com/rohitxdev/go-api-starter/cryptoutil"
	"github.com/rohitxdev/go-api-starter/email"
	"github.com/rohitxdev/go-api-starter/repo"
)

func (h *Handler) logOut(c echo.Context) error {
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

var authenticatedClients = make([]string, 0)

func (h *Handler) validateLogInToken(c echo.Context) error {
	req := new(logInRequest)
	if err := bindAndValidate(c, req); err != nil {
		return err
	}

	claims, err := auth.ValidateLoginToken(req.Token, h.Config.JWTSecret)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "Invalid token")
	}

	clientId := claims.ClientID
	if clientId != "" {
		c.SetCookie(&http.Cookie{
			Name:     "clientId",
			Value:    clientId,
			HttpOnly: true,
			MaxAge:   -1,
		})
		if !slices.Contains(authenticatedClients, clientId) {
			return echo.NewHTTPError(http.StatusUnauthorized, "Invalid token")
		}
		idx := slices.Index(authenticatedClients, clientId)
		authenticatedClients = append(authenticatedClients[:idx], authenticatedClients[idx+1:]...)

	} else {
		var user *repo.User
		if user, err = h.Repo.GetUserById(c.Request().Context(), claims.UserID); err != nil {
			return err
		}
		if !user.IsVerified {
			if err = h.Repo.SetIsVerified(c.Request().Context(), user.Id, true); err != nil {
				return err
			}
		}
	}
	if _, err = createSession(c, h.Config.SessionDuration, claims.UserID); err != nil {
		return err
	}
	return c.Render(http.StatusOK, "log-in-success.tmpl", nil)
}

type sendLoginEmailRequest struct {
	Email string `form:"email" json:"email" validate:"required,email"`
}

func (h *Handler) sendLoginEmail(c echo.Context) error {
	req := new(sendLoginEmailRequest)
	if err := bindAndValidate(c, req); err != nil {
		return err
	}

	host := c.Request().Host
	if host == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Host header is empty")
	}

	userEmail := sanitizeEmail(req.Email)
	var userID uint64
	user, err := h.Repo.GetUserByEmail(c.Request().Context(), userEmail)
	if user != nil {
		userID = user.Id
	} else {
		if !errors.Is(err, repo.ErrUserNotFound) {
			return fmt.Errorf("Failed to get user: %w", err)
		}
		userID, err = h.Repo.CreateUser(c.Request().Context(), userEmail)
		if err != nil {
			return fmt.Errorf("Failed to create user: %w", err)
		}
	}

	clientId := fingerprintUser(c.RealIP(), c.Request().UserAgent())
	c.SetCookie(&http.Cookie{
		Name:     "clientId",
		Value:    clientId,
		HttpOnly: true,
		MaxAge:   int(h.Config.LogInTokenExpiresIn.Seconds()),
	})

	token, err := auth.GenerateLoginToken(auth.TokenClaims{UserID: userID, ClientID: clientId}, h.Config.JWTSecret, h.Config.LogInTokenExpiresIn)
	if err != nil {
		return fmt.Errorf("Failed to generate login token: %w", err)
	}

	protocol := "http"
	if c.IsTLS() {
		protocol = "https"
	}
	emailData := map[string]any{
		"loginURL":     fmt.Sprintf("%s://%s%s?token=%s", protocol, host, c.Path(), token),
		"validMinutes": h.Config.LogInTokenExpiresIn.Minutes(),
	}
	emailOpts := email.BaseOpts{
		Subject:     "Log In to Your Account",
		ToAddresses: []string{req.Email},
		FromAddress: h.Config.SenderEmail,
		FromName:    "The App",
	}
	if err = h.Email.SendHTML(&emailOpts, "login.tmpl", emailData); err != nil {
		return fmt.Errorf("Failed to send email: %w", err)
	}

	return c.JSON(http.StatusOK, response{Message: "Login link sent to " + req.Email})
}
func fingerprintUser(IP string, userAgent string) string {
	return cryptoutil.Base62Hash(IP + userAgent)
}
