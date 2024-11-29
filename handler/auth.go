package handler

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/labstack/echo/v4"
	"github.com/rohitxdev/go-api-starter/cryptoutil"
	"github.com/rohitxdev/go-api-starter/email"
	"github.com/rohitxdev/go-api-starter/repo"
)

// @Summary Log out
// @Description Log out
// @Tags Auth
// @Accept json
// @Produce json
// @Success 200 {object} response
// @Failure 401 {object} response
// @Failure 500 {object} response
// @Router /auth/logout [get]
func (h *Handler) LogOut(c echo.Context) error {
	_, err := c.Cookie("refreshToken")
	if err != nil {
		return echo.ErrUnauthorized
	}
	clearAuthCookies(c)
	return c.JSON(http.StatusOK, response{Message: "Logged out successfully"})
}

// @Summary Log in
// @Description Log in
// @Tags Auth
// @Accept json
// @Produce json
// @Param email formData string true "Email"
// @Param password formData string true "Password"
// @Success 200 {object} response
// @Failure 400 {object} response
// @Failure 401 {object} response
// @Failure 500 {object} response
// @Router /auth/log-in [post]
func (h *Handler) LogIn(c echo.Context) error {
	var req struct {
		Email    string `form:"email" json:"email" validate:"required,email"`
		Password string `form:"password" json:"password" validate:"required"`
	}
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}
	req.Email = sanitizeEmail(req.Email)

	user, err := h.Repo.GetUserByEmail(c.Request().Context(), req.Email)
	if err != nil {
		return echo.ErrNotFound
	}

	if !cryptoutil.VerifyPassword(req.Password, user.PasswordHash) {
		return echo.ErrUnauthorized
	}

	if err = setAccessTokenCookie(c, h.Config.AccessTokenExpiresIn, user.ID, h.Config.AccessTokenSecret); err != nil {
		return fmt.Errorf("failed to set access token cookie: %w", err)
	}
	if err = setRefreshTokenCookie(c, h.Config.RefreshTokenExpiresIn, user.ID, h.Config.RefreshTokenSecret); err != nil {
		return fmt.Errorf("failed to set refresh token cookie: %w", err)
	}
	return c.JSON(http.StatusOK, response{Message: "Logged in successfully"})
}

// @Summary Sign up
// @Description Sign up
// @Tags Auth
// @Accept json
// @Produce json
// @Param email formData string true "Email"
// @Param password formData string true "Password"
// @Success 200 {object} response
// @Failure 400 {object} response
// @Failure 500 {object} response
// @Router /auth/signup [post]
func (h *Handler) SignUp(c echo.Context) error {
	var req struct {
		Email    string `form:"email" json:"email" validate:"required,email"`
		Password string `form:"password" json:"password" validate:"required"`
	}
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

	if err = setAccessTokenCookie(c, h.Config.AccessTokenExpiresIn, userID, h.Config.AccessTokenSecret); err != nil {
		return fmt.Errorf("failed to set access token cookie: %w", err)
	}
	if err = setRefreshTokenCookie(c, h.Config.RefreshTokenExpiresIn, userID, h.Config.RefreshTokenSecret); err != nil {
		return fmt.Errorf("failed to set refresh token cookie: %w", err)
	}
	return c.JSON(http.StatusCreated, response{Message: "Signed up successfully"})
}

// @Summary Get access token
// @Description Get access token
// @Tags Auth
// @Accept json
// @Produce json
// @Success 200 {object} response
// @Failure 401 {object} response
// @Failure 500 {object} response
// @Router /auth/access-token [get]
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

	userID, err := cryptoutil.VerifyJWT(refreshToken.Value, h.Config.RefreshTokenSecret)
	if err != nil {
		return echo.ErrUnauthorized
	}

	user, err := h.Repo.GetUserById(c.Request().Context(), userID)
	if err != nil {
		return echo.ErrNotFound
	}
	if user.AccountStatus != repo.AccountStatusActive {
		return echo.ErrForbidden
	}

	if err = setAccessTokenCookie(c, h.Config.AccessTokenExpiresIn, user.ID, h.Config.AccessTokenSecret); err != nil {
		return fmt.Errorf("failed to set access token cookie: %w", err)
	}
	return c.JSON(http.StatusOK, response{Message: "Generated access token successfully"})
}

// @Summary Change password
// @Description Change password
// @Tags Auth
// @Accept json
// @Produce json
// @Param currentPassword formData string true "Current password"
// @Param newPassword formData string true "New password"
// @Success 200 {object} response
// @Failure 401 {object} response
// @Failure 400 {object} response
// @Failure 500 {object} response
// @Router /auth/change-password [post]
func (h *Handler) ChangePassword(c echo.Context) error {
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
		return c.JSON(http.StatusUnauthorized, response{Message: "Incorrect password"})
	}
	if err := h.Repo.SetUserPassword(c.Request().Context(), user.ID, req.NewPassword); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, response{Message: "Password updated successfully"})
}

// @Summary Send reset password email
// @Description Send reset password email
// @Tags Auth
// @Accept json
// @Produce json
// @Param email formData string true "Email"
// @Param callbackURL formData string true "Callback URL"
// @Success 200 {object} response
// @Failure 400 {object} response
// @Failure 500 {object} response
// @Router /auth/reset-password [post]
func (h *Handler) SendResetPasswordEmail(c echo.Context) error {
	var req struct {
		Email       string `form:"email" json:"email" validate:"required,email"`
		CallbackURL string `form:"callbackUrl" json:"callbackUrl" validate:"required"`
	}
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	user, err := h.Repo.GetUserByEmail(c.Request().Context(), sanitizeEmail(req.Email))
	if err != nil {
		return echo.ErrNotFound
	}
	resetToken, err := cryptoutil.GenerateJWT(user.ID, h.Config.CommonTokenExpiresIn, h.Config.CommonTokenSecret)
	if err != nil {
		return err
	}

	emailOpts := email.BaseOpts{
		Subject:     "Reset your password",
		FromAddress: h.Config.SenderEmail,
		FromName:    h.Config.AppName,
		ToAddresses: []string{req.Email},
	}
	callbackURL, err := url.Parse(req.CallbackURL)
	if err != nil {
		return err
	}
	query := callbackURL.Query()
	query.Add("token", resetToken)
	callbackURL.RawQuery = query.Encode()

	data := map[string]any{
		"callbackURL":  callbackURL.String(),
		"validMinutes": h.Config.CommonTokenExpiresIn.Minutes(),
	}
	if err = h.Email.SendHTML(&emailOpts, "reset-password", data); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, response{Message: "Sent password reset email successfully"})
}

// @Summary Reset password
// @Description Reset password
// @Tags Auth
// @Accept json
// @Produce json
// @Param token formData string true "Token"
// @Param newPassword formData string true "New password"
// @Success 200 {object} response
// @Failure 400 {object} response
// @Failure 500 {object} response
// @Router /auth/reset-password [get]
func (h *Handler) ResetPassword(c echo.Context) error {
	var req struct {
		Token       string `query:"token" validate:"required"`
		NewPassword string `query:"newPassword" validate:"required,min=8,max=128"`
	}
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	userID, err := cryptoutil.VerifyJWT(req.Token, h.Config.CommonTokenSecret)
	if err != nil {
		return echo.ErrUnauthorized
	}
	if _, err = h.Repo.GetUserById(c.Request().Context(), userID); err != nil {
		return echo.ErrNotFound
	}
	passwordHash, err := cryptoutil.HashPassword(req.NewPassword)
	if err != nil {
		return err
	}
	if err = h.Repo.SetUserPassword(c.Request().Context(), userID, passwordHash); err != nil {
		return err
	}
	return c.Render(http.StatusOK, "reset-password-success", nil)
}

// @Summary Send account verification email
// @Description Send account verification email
// @Tags Auth
// @Accept json
// @Produce json
// @Param email formData string true "Email"
// @Success 200 {object} response
// @Failure 400 {object} response
// @Failure 500 {object} response
// @Router /auth/verify-account [post]
func (h *Handler) SendAccountVerificationEmail(c echo.Context) error {
	var req struct {
		Email       string `form:"email" json:"email" validate:"required,email"`
		CallbackURL string `form:"callbackUrl" json:"callbackUrl" validate:"required"`
	}
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	user, err := h.Repo.GetUserByEmail(c.Request().Context(), sanitizeEmail(req.Email))
	if err != nil {
		return echo.ErrNotFound
	}
	if user.AccountStatus != repo.AccountStatusPending {
		return c.JSON(http.StatusBadRequest, response{Message: "Account status is not PENDING"})
	}
	verificationToken, err := cryptoutil.GenerateJWT(user.ID, h.Config.CommonTokenExpiresIn, h.Config.CommonTokenSecret)
	if err != nil {
		return err
	}

	emailOpts := email.BaseOpts{
		Subject:     "Verify your account",
		FromAddress: h.Config.SenderEmail,
		FromName:    "Go App",
		ToAddresses: []string{req.Email},
	}
	callbackURL, err := url.Parse(req.CallbackURL)
	if err != nil {
		return err
	}
	query := callbackURL.Query()
	query.Add("token", verificationToken)
	callbackURL.RawQuery = query.Encode()

	data := map[string]any{
		"callbackURL":  callbackURL.String(),
		"validMinutes": h.Config.CommonTokenExpiresIn.Minutes(),
	}
	if err = h.Email.SendHTML(&emailOpts, "verify-account", data); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, response{Message: "Sent email verification email successfully"})
}

// @Summary Verify account
// @Description Verify account
// @Tags Auth
// @Accept json
// @Produce json
// @Param token formData string true "Token"
// @Success 200 {object} response
// @Failure 400 {object} response
// @Failure 500 {object} response
// @Router /auth/verify-account [get]
func (h *Handler) VerifyAccount(c echo.Context) error {
	var req struct {
		Token string `query:"token" validate:"required"`
	}
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	userID, err := cryptoutil.VerifyJWT(req.Token, h.Config.CommonTokenSecret)
	if err != nil {
		return echo.ErrUnauthorized
	}
	user, err := h.Repo.GetUserById(c.Request().Context(), userID)
	if err != nil {
		return echo.ErrUnauthorized
	}
	if user.AccountStatus != repo.AccountStatusPending {
		return c.JSON(http.StatusBadRequest, response{Message: "Account status is not PENDING"})
	}
	if err = h.Repo.SetAccountStatusActive(c.Request().Context(), user.ID); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, response{Message: "Account verified successfully"})
}
