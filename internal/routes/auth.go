package routes

import (
	"errors"
	"fmt"
	"net/http"
	"slices"
	"time"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/rohitxdev/go-api-starter/internal/auth"
	"github.com/rohitxdev/go-api-starter/internal/cryptoutil"
	"github.com/rohitxdev/go-api-starter/internal/email"
	"github.com/rohitxdev/go-api-starter/internal/repo"
)

const (
	logInTokenExpiresIn = time.Minute * 10 // 10 minutes
)

func LogOut(svc *Services) echo.HandlerFunc {
	return func(c echo.Context) error {
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
}

type logInRequest struct {
	Token string `query:"token" validate:"required"`
}

var authenticatedClients = make([]string, 0)

func ValidateLogInToken(svc *Services) echo.HandlerFunc {
	return func(c echo.Context) error {
		req := new(logInRequest)
		if err := bindAndValidate(c, req); err != nil {
			return err
		}

		claims, err := auth.ValidateLoginToken(req.Token, svc.Config.JWTSecret)
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
			if user, err = svc.Repo.GetUserById(c.Request().Context(), claims.UserID); err != nil {
				return err
			}
			if !user.IsVerified {
				if err = svc.Repo.SetIsVerified(c.Request().Context(), user.Id, true); err != nil {
					return err
				}
			}
		}
		if _, err = createSession(c, claims.UserID); err != nil {
			return err
		}
		return c.Render(http.StatusOK, "log-in-success.tmpl", nil)
	}
}

type sendLoginEmailRequest struct {
	Email string `form:"email" json:"email" validate:"required,email"`
}

func SendLoginEmail(svc *Services) echo.HandlerFunc {
	return func(c echo.Context) error {
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
		user, err := svc.Repo.GetUserByEmail(c.Request().Context(), userEmail)
		if user != nil {
			userID = user.Id
		} else {
			if !errors.Is(err, repo.ErrUserNotFound) {
				return fmt.Errorf("Failed to get user: %w", err)
			}
			userID, err = svc.Repo.CreateUser(c.Request().Context(), userEmail)
			if err != nil {
				return fmt.Errorf("Failed to create user: %w", err)
			}
		}

		clientId := fingerprintUser(c.RealIP(), c.Request().UserAgent())
		c.SetCookie(&http.Cookie{
			Name:     "clientId",
			Value:    clientId,
			HttpOnly: true,
			MaxAge:   int(logInTokenExpiresIn.Seconds()),
		})

		token, err := auth.GenerateLoginToken(auth.TokenClaims{UserID: userID, ClientID: clientId}, svc.Config.JWTSecret, logInTokenExpiresIn)
		if err != nil {
			return fmt.Errorf("Failed to generate login token: %w", err)
		}

		emailHeaders := &email.Headers{
			Subject:     "Log In to Your Account",
			ToAddresses: []string{req.Email},
			FromAddress: svc.Config.SenderEmail,
			FromName:    "The App",
		}
		protocol := "http"
		if c.IsTLS() {
			protocol = "https"
		}
		emailData := map[string]any{
			"loginURL":     fmt.Sprintf("%s://%s%s?token=%s", protocol, host, c.Path(), token),
			"validMinutes": logInTokenExpiresIn.Minutes(),
		}
		if err = svc.Email.SendHTML(emailHeaders, "login.tmpl", emailData); err != nil {
			return fmt.Errorf("Failed to send email: %w", err)
		}
		return c.JSON(http.StatusOK, response{Message: "Login link sent to " + req.Email})
	}
}

func fingerprintUser(IP string, userAgent string) string {
	return cryptoutil.Base62Hash(IP + userAgent)
}
