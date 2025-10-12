package handler

import (
	"fmt"
	"net/http"
	"net/netip"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/rohitxdev/go-api/database/repository"
	"github.com/rohitxdev/go-api/util"
)

func (h *Handler) SendAuthOTP(c echo.Context) error {
	var req struct {
		Email string `json:"email" validate:"required,email"`
	}
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	req.Email = canonicalizeEmail(req.Email)
	code, err := util.GenerateAlphaNumCode(6)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: "failed to generate OTP",
		})
	}
	user, err := h.Repo.UpsertUser(c.Request().Context(), req.Email)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Errorf("failed to upsert user: %w", err))
	}

	codeHash, err := util.GenerateSecureHash([]byte(code))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Errorf("failed to hash security code: %w", err))
	}

	if err = h.Repo.CreateOtp(c.Request().Context(), repository.CreateOtpParams{
		UserID:   user.ID,
		CodeHash: codeHash,
		ExpiresAt: pgtype.Timestamptz{
			Time:  time.Now().Add(time.Minute * 10),
			Valid: true,
		},
	}); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Errorf("failed to create OTP: %w", err))
	}

	return c.JSON(http.StatusOK, APIResponseWithPayload{
		Success: true,
		Message: "sign in OTP sent successfully",
		Payload: echo.Map{
			"userId": user.ID,
			"code":   code,
		},
	})
}

func (h *Handler) VerifyAuthOTP(c echo.Context) error {
	var req struct {
		UserID pgtype.UUID `json:"user_id" validate:"required,uuid"`
		Code   string      `json:"code" validate:"required,len=6"`
	}
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	otp, err := h.Repo.GetOtpByUserId(c.Request().Context(), req.UserID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, APIResponse{
			Message: "otp not found or invalid",
		})
	}
	if otp.Attempts > 3 {
		if err = h.Repo.DeleteOtp(c.Request().Context(), otp.ID); err != nil {
			return serverError(fmt.Errorf("failed to delete OTP: %w", err))
		}
		return c.JSON(http.StatusForbidden, APIResponse{
			Message: "max attempts exceeded",
		})
	}

	if err = h.Repo.IncrementOtpAttempts(c.Request().Context(), req.UserID); err != nil {
		return serverError(fmt.Errorf("failed to increment OTP attempts: %w", err))
	}

	if !util.VerifySecureHash([]byte(req.Code), otp.CodeHash) {
		return c.JSON(http.StatusBadRequest, APIResponse{
			Message: "invalid OTP code",
		})
	}

	if err = h.Repo.DeleteOtp(c.Request().Context(), otp.ID); err != nil {
		return serverError(fmt.Errorf("failed to delete OTP: %w", err))
	}

	if _, err = h.Repo.UpdateUser(
		c.Request().Context(),
		repository.UpdateUserParams{
			ID: otp.UserID,
			VerifiedAt: pgtype.Timestamptz{
				Time:  time.Now(),
				Valid: true,
			},
		}); err != nil {
		return serverError(fmt.Errorf("failed to update user: %w", err))
	}

	ipAddress, err := netip.ParseAddr(c.RealIP())
	if err != nil {
		return serverError(fmt.Errorf("failed to parse client IP address: %w", err))
	}

	sessionId, err := h.Repo.CreateSession(
		c.Request().Context(),
		repository.CreateSessionParams{
			UserID: otp.UserID,
			ExpiresAt: pgtype.Timestamptz{
				Time:  time.Now().Add(time.Hour * 24 * 30),
				Valid: true,
			},
			IpAddress: ipAddress,
		})
	if err != nil {
		return serverError(fmt.Errorf("failed to create session: %w", err))
	}

	sess, err := session.Get("session", c)
	if err != nil {
		return serverError(fmt.Errorf("failed to get session: %w", err))
	}

	sess.Values["session_id"] = sessionId.String()
	if err = sess.Save(c.Request(), c.Response()); err != nil {
		return serverError(fmt.Errorf("failed to get session: %w", err))
	}

	return c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Message: "OTP verified successfully",
	})
}

func (h *Handler) SignOut(c echo.Context) error {
	sess, err := session.Get("session", c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, APIResponse{
			Message: "user not signed in",
		})
	}

	sess.Options.MaxAge = -1

	if err = sess.Save(c.Request(), c.Response()); err != nil {
		return serverError(fmt.Errorf("failed to save session: %w", err))
	}

	return c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Message: "logged out successfully",
	})
}
