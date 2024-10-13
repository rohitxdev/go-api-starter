package handler

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/rohitxdev/go-api-starter/internal/repo"
	"golang.org/x/crypto/bcrypt"
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

func (h *handler) LogOut(c echo.Context) error {
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
	Email    string `form:"email" json:"email" validate:"required,email"`
	Password string `form:"password" json:"password" validate:"required"`
}

func (h *handler) LogIn(c echo.Context) error {
	req := new(logInRequest)
	if err := bindAndValidate(c, req); err != nil {
		return err
	}
	user, err := h.repo.GetUserByEmail(c.Request().Context(), sanitizeEmail(req.Email))
	if err != nil {
		return err
	}
	if err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return c.String(http.StatusUnauthorized, err.Error())
	}
	if _, err := createSession(c, user.Id); err != nil {
		return err
	}
	return c.String(http.StatusOK, "Logged in successfully")
}

type signUpRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8,max=64"`
}

func (h *handler) SignUp(c echo.Context) error {
	req := new(signUpRequest)
	if err := bindAndValidate(c, req); err != nil {
		return err
	}
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), 12)
	if err != nil {
		return err
	}
	user := &repo.UserCore{
		Email:        sanitizeEmail(req.Email),
		PasswordHash: string(passwordHash),
	}
	userId, err := h.repo.CreateUser(c.Request().Context(), user)
	if err != nil {
		fmt.Println(err)
		return c.String(http.StatusBadRequest, err.Error())
	}
	if _, err := createSession(c, userId); err != nil {
		return err
	}
	return c.String(http.StatusCreated, "Signed up successfully")
}

type changePasswordRequest struct {
	CurrentPassword string `json:"currentPassword" validate:"required,min=8,max=64"`
	NewPassword     string `json:"newPassword" validate:"required,min=8,max=64"`
}

func (h *handler) ChangePassword(c echo.Context) error {
	req := new(changePasswordRequest)
	if err := bindAndValidate(c, req); err != nil {
		return err
	}
	sess, err := session.Get("session", c)
	if err != nil {
		return c.String(http.StatusUnauthorized, ErrUserNotLoggedIn.Error())
	}
	userId, ok := sess.Values["user_id"].(string)
	if !ok {
		return c.String(http.StatusUnauthorized, ErrUserNotLoggedIn.Error())
	}
	user, err := h.repo.GetUserById(c.Request().Context(), userId)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}
	if err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.CurrentPassword)); err != nil {
		return c.String(http.StatusUnauthorized, err.Error())
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte(req.NewPassword), 12)
	err = h.repo.Update(c.Request().Context(), userId, map[string]any{
		"password_hash": string(hash),
	})
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}
	return c.String(http.StatusOK, "Password changed successfully")
}
