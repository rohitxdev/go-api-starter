package handler

import (
	"time"

	"github.com/labstack/echo-contrib/echoprometheus"
	"github.com/labstack/echo/v4"
	echoSwagger "github.com/swaggo/echo-swagger"
)

func mountRoutes(e *echo.Echo, h *Handler) {
	limit := rateLimiter(!h.Config.IsDev)

	e.Pre(limit(100, time.Minute))
	e.GET("/metrics", echoprometheus.NewHandler())
	e.GET("/swagger/*", echoSwagger.EchoWrapHandler())
	// @Summary Get client config
	// @Success 200 {object} ResponseWithPayload[ClientConfig]
	// @Router /config [get]
	e.GET("/config", h.ClientConfig)
	// @Summary Home Page
	// @Success 200 {html} string "home page"
	// @Router / [get]
	e.GET("/", h.Home)

	auth := e.Group("/auth")
	{
		// @Summary Sign up
		// @Tags Auth
		// @Accept json
		// @Produce json
		// @Param email formData string true "Email"
		// @Param password formData string true "Password"
		// @Success 200 {object} Response
		// @Router /auth/sign-up [post]
		auth.POST("/sign-up", h.SignUp)
		// @Summary Log in
		// @Tags Auth
		// @Accept json
		// @Produce json
		// @Param email formData string true "Email"
		// @Param password formData string true "Password"
		// @Success 200 {object} Response
		// @Router /auth/log-in [post]
		auth.POST("/log-in", h.LogIn)
		// @Summary Log out
		// @Tags Auth
		// @Accept json
		// @Produce json
		// @Success 200 {object} Response
		// @Router /auth/log-out [get]
		auth.GET("/log-out", h.LogOut)
		// @Summary Get access token
		// @Tags Auth
		// @Accept json
		// @Produce json
		// @Success 200 {object} response
		// @Router /auth/access-token [get]
		auth.GET("/access-token", h.AccessToken, limit(2, h.Config.AccessTokenExpiresIn))
		user := auth.Group("/user")
		{
			// @Summary Get current user
			// @Tags Auth
			// @Accept json
			// @Produce json
			// @Security ApiKeyAuth
			// @Success 200 {object} ResponseWithPayload[repo.PublicUser]
			// @Router /auth/user [get]
			user.GET("", h.CurrentUser)
			// @Summary Update password
			// @Tags Auth
			// @Accept json
			// @Produce json
			// @Param currentPassword formData string true "Current password"
			// @Param newPassword formData string true "New password"
			// @Success 200 {object} Response
			// @Router /auth/password [put]
			password := user.Group("/password")
			{
				password.PUT("", h.UpdatePassword)
				// @Summary Send reset password email
				// @Tags Auth
				// @Accept json
				// @Produce json
				// @Param email formData string true "Email"
				// @Param callbackUrl formData string true "Callback URL"
				// @Success 200 {object} Response
				// @Router /auth/password/reset [post]
				password.POST("/reset", h.SendResetPasswordEmail, limit(2, time.Minute))
				// @Summary Reset password
				// @Tags Auth
				// @Accept json
				// @Produce json
				// @Param token formData string true "Token"
				// @Param newPassword formData string true "New password"
				// @Success 200 {object} Response
				// @Router /auth/password/reset [put]
				password.PUT("/reset", h.ResetPassword)
			}
		}
		verification := auth.Group("/verify")
		{
			// @Summary Send account verification email
			// @Tags Auth
			// @Accept json
			// @Produce json
			// @Param email formData string true "Email"
			// @Param callbackUrl formData string true "Callback URL"
			// @Success 200 {object} Response
			// @Router /auth/verify/account [post]
			verification.POST("/account", h.SendAccountVerificationEmail, limit(2, time.Minute))
			// @Summary Verify account
			// @Tags Auth
			// @Accept json
			// @Produce json
			// @Param token formData string true "Token"
			// @Success 200 {object} Response
			// @Router /auth/verify/account [put]
			verification.PUT("/account", h.VerifyAccount)
		}
	}
}
