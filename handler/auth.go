package handler

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/rohitxdev/go-api-starter/cryptoutil"
	"github.com/rohitxdev/go-api-starter/repo"
)

func (h *Handler) LogOut(c echo.Context) error {
	_, err := c.Cookie("refreshToken")
	if err != nil {
		return echo.ErrUnauthorized
	}
	clearAuthCookies(c)
	return c.JSON(http.StatusOK, response{Message: "Logged out successfully"})
}

type logInRequest struct {
	Email    string `form:"email" json:"email" validate:"required,email"`
	Password string `form:"password" json:"password" validate:"required"`
}

func (h *Handler) LogIn(c echo.Context) error {
	var req logInRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}
	req.Email = sanitizeEmail(req.Email)

	user, err := h.Repo.GetUserByEmail(c.Request().Context(), req.Email)
	if err != nil {
		if errors.Is(err, repo.ErrUserNotFound) {
			return echo.ErrUnauthorized
		}
		return err
	}

	if !cryptoutil.VerifyPassword(req.Password, user.PasswordHash) {
		return echo.ErrUnauthorized
	}

	if err = setAccessTokenCookie(c, h.Config.AccessTokenExpiresIn, user.ID, h.Config.JWTSecret); err != nil {
		return fmt.Errorf("failed to set access token cookie: %w", err)
	}
	if err = setRefreshTokenCookie(c, h.Config.RefreshTokenExpiresIn, user.ID, h.Config.JWTSecret); err != nil {
		return fmt.Errorf("failed to set refresh token cookie: %w", err)
	}
	return c.JSON(http.StatusOK, response{Message: "Logged in successfully"})
}

type signUpRequest struct {
	Email    string `form:"email" json:"email" validate:"required,email"`
	Password string `form:"password" json:"password" validate:"required"`
}

func (h *Handler) SignUp(c echo.Context) error {
	var req signUpRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}
	req.Email = sanitizeEmail(req.Email)

	var userID uint64
	_, err := h.Repo.GetUserByEmail(c.Request().Context(), req.Email)
	if err == nil {
		return c.JSON(http.StatusBadRequest, response{Message: "User already exists"})
	}

	passwordHash, err := cryptoutil.HashPassword(req.Password)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	if userID, err = h.Repo.CreateUser(c.Request().Context(), req.Email, passwordHash); err != nil {
		switch err {
		case repo.ErrUserAlreadyExists:
			return c.JSON(http.StatusBadRequest, response{Message: "User already exists"})
		default:
			return fmt.Errorf("failed to create user: %w", err)
		}
	}

	if err = setAccessTokenCookie(c, h.Config.AccessTokenExpiresIn, userID, h.Config.JWTSecret); err != nil {
		return fmt.Errorf("failed to set access token cookie: %w", err)
	}
	if err = setRefreshTokenCookie(c, h.Config.RefreshTokenExpiresIn, userID, h.Config.JWTSecret); err != nil {
		return fmt.Errorf("failed to set refresh token cookie: %w", err)
	}
	return c.JSON(http.StatusCreated, response{Message: "Signed up successfully"})
}

func (h *Handler) GetAccessToken(c echo.Context) error {
	refreshToken, err := c.Cookie("refreshToken")
	if err != nil {
		return echo.ErrUnauthorized
	}
	defer func() {
		if err != nil {
			clearAuthCookies(c)
		}
	}()

	userID, err := cryptoutil.VerifyJWT(refreshToken.Value, h.Config.JWTSecret)
	if err != nil {
		return echo.ErrUnauthorized
	}

	user, err := h.Repo.GetUserById(c.Request().Context(), userID)
	if err != nil {
		return echo.ErrUnauthorized
	}
	if user.AccountStatus != repo.AccountStatusActive {
		return echo.ErrForbidden
	}

	if err = setAccessTokenCookie(c, h.Config.AccessTokenExpiresIn, user.ID, h.Config.JWTSecret); err != nil {
		return fmt.Errorf("failed to set access token cookie: %w", err)
	}
	return c.JSON(http.StatusOK, response{Message: "Generated access token successfully"})
}
