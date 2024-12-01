package handler

import (
	"fmt"
	"net/http"
	"net/url"
	"slices"

	"github.com/labstack/echo/v4"
	"github.com/rohitxdev/go-api-starter/cryptoutil"
	"github.com/rohitxdev/go-api-starter/email"
	"github.com/rohitxdev/go-api-starter/repo"
)

func (h *Handler) SignUp(c echo.Context) error {
	var req struct {
		Email    string `form:"email" json:"email" validate:"required,email"`
		Password string `form:"password" json:"password" validate:"required,min=8,max=128"`
	}
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}
	req.Email = canonicalizeEmail(req.Email)

	var userID uint64
	_, err := h.Repo.GetUserByEmail(c.Request().Context(), req.Email)
	if err == nil {
		return c.JSON(http.StatusBadRequest, Response{Message: "User already exists"})
	}

	passwordHash, err := cryptoutil.HashPassword(req.Password)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	if userID, err = h.Repo.CreateUser(c.Request().Context(), req.Email, passwordHash); err != nil {
		switch err {
		case repo.ErrUserAlreadyExists:
			return c.JSON(http.StatusBadRequest, Response{Message: "User already exists"})
		default:
			return fmt.Errorf("failed to create user: %w", err)
		}
	}

	if err = setAccessTokenCookie(c, h.Config.AccessTokenExpiresIn, userID, h.Config.AccessTokenSecret); err != nil {
		return fmt.Errorf("failed to set access token cookie: %w", err)
	}
	if err = setRefreshTokenCookie(c, h.Config.RefreshTokenExpiresIn, userID, h.Config.RefreshTokenSecret); err != nil {
		return fmt.Errorf("failed to set refresh token cookie: %w", err)
	}
	return c.JSON(http.StatusOK, Response{Message: "Signed up successfully", Success: true})
}

func (h *Handler) LogIn(c echo.Context) error {
	var req struct {
		Email    string `form:"email" json:"email" validate:"required,email"`
		Password string `form:"password" json:"password" validate:"required,min=8,max=128"`
	}
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}
	req.Email = canonicalizeEmail(req.Email)

	user, err := h.Repo.GetUserByEmail(c.Request().Context(), req.Email)
	if err != nil {
		return c.JSON(http.StatusNotFound, Response{Message: "User not found"})
	}

	if !cryptoutil.VerifyPassword(req.Password, user.PasswordHash) {
		return c.JSON(http.StatusUnauthorized, Response{Message: "Incorrect password"})
	}

	if err = setAccessTokenCookie(c, h.Config.AccessTokenExpiresIn, user.ID, h.Config.AccessTokenSecret); err != nil {
		return fmt.Errorf("failed to set access token cookie: %w", err)
	}
	if err = setRefreshTokenCookie(c, h.Config.RefreshTokenExpiresIn, user.ID, h.Config.RefreshTokenSecret); err != nil {
		return fmt.Errorf("failed to set refresh token cookie: %w", err)
	}
	return c.JSON(http.StatusOK, Response{Message: "Logged in successfully", Success: true})
}

func (h *Handler) LogOut(c echo.Context) error {
	_, err := c.Cookie("refreshToken")
	if err != nil {
		return c.JSON(http.StatusBadRequest, Response{Message: "User is not logged in"})
	}
	clearAuthCookies(c)
	return c.JSON(http.StatusOK, Response{Message: "Logged out successfully", Success: true})
}

func (h *Handler) GetAccessToken(c echo.Context) error {
	refreshToken, err := c.Cookie("refreshToken")
	if err != nil {
		return c.JSON(http.StatusUnauthorized, Response{Message: "User is not logged in"})
	}
	defer func() {
		if err != nil {
			clearAuthCookies(c)
		}
	}()

	userID, err := cryptoutil.VerifyJWT(refreshToken.Value, h.Config.RefreshTokenSecret)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, Response{Message: "Refresh token verification failed"})
	}

	user, err := h.Repo.GetUserById(c.Request().Context(), userID)
	if err != nil {
		return c.JSON(http.StatusNotFound, Response{Message: "User not found"})
	}
	if user.AccountStatus != repo.AccountStatusActive {
		return c.JSON(http.StatusForbidden, Response{Message: "Account status is not ACTIVE"})
	}

	if err = setAccessTokenCookie(c, h.Config.AccessTokenExpiresIn, user.ID, h.Config.AccessTokenSecret); err != nil {
		return fmt.Errorf("failed to set access token cookie: %w", err)
	}
	return c.JSON(http.StatusOK, Response{Message: "Generated access token successfully", Success: true})
}

func (h *Handler) UpdatePassword(c echo.Context) error {
	user, err := h.checkAuth(c, RoleUser)
	if err != nil {
		return err
	}
	var req struct {
		CurrentPassword string `form:"currentPassword" json:"currentPassword" validate:"required"`
		NewPassword     string `form:"newPassword" json:"newPassword" validate:"required,min=8,max=128"`
	}
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	if !cryptoutil.VerifyPassword(req.CurrentPassword, user.PasswordHash) {
		return c.JSON(http.StatusUnauthorized, Response{Message: "Incorrect password"})
	}
	if err := h.Repo.SetUserPassword(c.Request().Context(), user.ID, req.NewPassword); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, Response{Message: "Password updated successfully", Success: true})
}

func (h *Handler) SendResetPasswordEmail(c echo.Context) error {
	var req struct {
		Email       string `form:"email" json:"email" validate:"required,email"`
		CallbackURL string `form:"callbackUrl" json:"callbackUrl" validate:"required"`
	}
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	user, err := h.Repo.GetUserByEmail(c.Request().Context(), canonicalizeEmail(req.Email))
	if err != nil {
		return c.JSON(http.StatusNotFound, Response{Message: "User not found"})
	}
	resetToken, err := cryptoutil.GenerateJWT(user.ID, h.Config.CommonTokenExpiresIn, h.Config.CommonTokenSecret)
	if err != nil {
		return err
	}

	callbackURL, err := url.Parse(req.CallbackURL)
	if err != nil {
		return err
	}

	if !h.Config.IsDev && !slices.Contains(h.Config.AllowedOrigins, callbackURL.Hostname()) {
		return c.JSON(http.StatusBadRequest, Response{Message: "Unauthorized origin for callback URL"})
	}

	query := callbackURL.Query()
	query.Add("token", resetToken)
	callbackURL.RawQuery = query.Encode()

	emailOpts := email.BaseOpts{
		Subject:     "Reset your password",
		FromAddress: h.Config.SenderEmail,
		FromName:    h.Config.AppName,
		ToAddresses: []string{req.Email},
		NoStack:     true,
	}
	data := map[string]any{
		"callbackURL":  callbackURL.String(),
		"validMinutes": h.Config.CommonTokenExpiresIn.Minutes(),
	}
	if err = h.Email.SendHTML(&emailOpts, "reset-password", data); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, Response{Message: "Sent password reset email successfully", Success: true})
}

func (h *Handler) ResetPassword(c echo.Context) error {
	var req struct {
		Token       string `form:"token" json:"token" validate:"required"`
		NewPassword string `form:"newPassword" json:"newPassword" validate:"required,min=8,max=128"`
	}
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	userID, err := cryptoutil.VerifyJWT(req.Token, h.Config.CommonTokenSecret)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, Response{Message: "JWT verification failed"})
	}
	if _, err = h.Repo.GetUserById(c.Request().Context(), userID); err != nil {
		return c.JSON(http.StatusNotFound, Response{Message: "User not found"})
	}
	passwordHash, err := cryptoutil.HashPassword(req.NewPassword)
	if err != nil {
		return err
	}
	if err = h.Repo.SetUserPassword(c.Request().Context(), userID, passwordHash); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, Response{Message: "Password updated successfully", Success: true})
}

func (h *Handler) SendAccountVerificationEmail(c echo.Context) error {
	var req struct {
		Email       string `form:"email" json:"email" validate:"required,email"`
		CallbackURL string `form:"callbackUrl" json:"callbackUrl" validate:"required"`
	}
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	user, err := h.Repo.GetUserByEmail(c.Request().Context(), canonicalizeEmail(req.Email))
	if err != nil {
		return c.JSON(http.StatusNotFound, Response{Message: "User not found"})
	}
	if user.AccountStatus != repo.AccountStatusPending {
		return c.JSON(http.StatusBadRequest, Response{Message: "Account status is not PENDING"})
	}
	verificationToken, err := cryptoutil.GenerateJWT(user.ID, h.Config.CommonTokenExpiresIn, h.Config.CommonTokenSecret)
	if err != nil {
		return err
	}

	callbackURL, err := url.Parse(req.CallbackURL)
	if err != nil {
		return err
	}

	if !h.Config.IsDev && !slices.Contains(h.Config.AllowedOrigins, callbackURL.Hostname()) {
		return c.JSON(http.StatusBadRequest, Response{Message: "Unauthorized origin for callback URL"})
	}

	query := callbackURL.Query()
	query.Add("token", verificationToken)
	callbackURL.RawQuery = query.Encode()

	emailOpts := email.BaseOpts{
		Subject:     "Verify your account",
		FromAddress: h.Config.SenderEmail,
		FromName:    h.Config.AppName,
		ToAddresses: []string{req.Email},
		NoStack:     true,
	}
	data := map[string]any{
		"callbackURL":  callbackURL.String(),
		"validMinutes": h.Config.CommonTokenExpiresIn.Minutes(),
	}
	if err = h.Email.SendHTML(&emailOpts, "verify-account", data); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, Response{Message: "Sent verification email successfully", Success: true})
}

func (h *Handler) VerifyAccount(c echo.Context) error {
	var req struct {
		Token string `form:"token" json:"token" validate:"required"`
	}
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	userID, err := cryptoutil.VerifyJWT(req.Token, h.Config.CommonTokenSecret)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, Response{Message: "JWT verification failed"})
	}
	user, err := h.Repo.GetUserById(c.Request().Context(), userID)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, Response{Message: "User not found"})
	}
	if user.AccountStatus != repo.AccountStatusPending {
		return c.JSON(http.StatusBadRequest, Response{Message: "Account status is not PENDING"})
	}
	if err = h.Repo.SetAccountStatusActive(c.Request().Context(), user.ID); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, Response{Message: "Account verified successfully", Success: true})
}

func (h *Handler) GetCurrentUser(c echo.Context) error {
	user, err := h.checkAuth(c, RoleUser)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, ResponseWithPayload[repo.PublicUser]{
		Message: "Fetched current user successfully",
		Success: true,
		Payload: repo.PublicUser{
			ID:          user.ID,
			Role:        user.Role,
			Email:       user.Email,
			Username:    user.Username,
			ImageURL:    user.ImageURL,
			Gender:      user.Gender,
			DateOfBirth: user.DateOfBirth,
			CreatedAt:   user.CreatedAt,
			UpdatedAt:   user.UpdatedAt,
		},
	})
}
