package handler

import (
	"net"
	"time"

	"github.com/labstack/echo-contrib/echoprometheus"
	"github.com/labstack/echo/v4"
	"github.com/rohitxdev/go-api-starter/docs"
	echoSwagger "github.com/swaggo/echo-swagger"
)

// @title Go API Starter
// @version 1.0.0
// @description Go API Starter is a boilerplate for building RESTful APIs in Go.
// @BasePath /
func mountRoutes(e *echo.Echo, h *Handler) {
	docs.SwaggerInfo.Host = net.JoinHostPort(h.Config.Host, h.Config.Port)
	limit := rateLimiter(!h.Config.IsDev)

	e.Pre(limit(100, time.Minute))
	e.GET("/metrics", echoprometheus.NewHandler())
	e.GET("/swagger/*", echoSwagger.EchoWrapHandler())
	e.GET("/config", h.ClientConfig)
	e.GET("/", h.Home)

	auth := e.Group("/auth")
	{
		auth.POST("/sign-up", h.SignUp)
		auth.POST("/log-in", h.LogIn)
		auth.GET("/log-out", h.LogOut)
		auth.GET("/access-token", h.AccessToken, limit(2, h.Config.AccessTokenExpiresIn))
		user := auth.Group("/user")
		{
			user.GET("", h.User)
			user.DELETE("", h.DeleteUser)
			password := user.Group("/password")
			{
				password.PUT("", h.UpdatePassword)
				password.POST("/reset", h.SendResetPasswordEmail, limit(2, time.Minute))
				password.PUT("/reset", h.ResetPassword)
			}
		}
		verification := auth.Group("/verify")
		{
			verification.POST("/email", h.SendAccountVerificationEmail, limit(2, time.Minute))
			verification.PUT("/email", h.VerifyEmail)
		}
	}
}
