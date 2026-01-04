package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/rohitxdev/go-api/deps/email"
	"github.com/rohitxdev/go-api/util"
)

const (
	redisPrefixSignInVerificationCode = "verification_code:sign_in:"
)

func (h *Handler) SendSignInVerificationCode(c echo.Context) error {
	var input struct {
		Email string `json:"email" validate:"required,email"`
	}
	if err := bindAndValidate(c, &input); err != nil {
		return err
	}

	input.Email = strings.ToLower(input.Email)
	code, err := util.GenerateAlphaNumCode(6)
	if err != nil {
		return fmt.Errorf("failed to generate verification code: %w", err)
	}

	codeHash, err := util.GenerateSecureHash([]byte(code))
	if err != nil {
		return fmt.Errorf("failed to hash verification code: %w", err)
	}

	ctx := c.Request().Context()
	cfg := h.Config.Get()
	key := redisPrefixSignInVerificationCode + input.Email

	tx := h.Redis.TxPipeline()
	tx.HSet(ctx, key, "hash", codeHash, "attempts", 0)
	tx.Expire(ctx, key, cfg.VerificationCodeTTL)
	if _, err := tx.Exec(ctx); err != nil {
		return fmt.Errorf("failed to save verification code to redis: %w", err)
	}

	emailOpts := email.BaseOpts{
		Subject:     "Your Verification Code",
		FromName:    cfg.EmailFromName,
		FromAddress: cfg.EmailFromAddress,
		ToAddresses: []string{input.Email},
	}
	if err = h.Email.SendHTML(
		&emailOpts,
		"sign-in-verification",
		map[string]any{
			"code":       code,
			"ttlMinutes": int(cfg.VerificationCodeTTL.Minutes()),
		},
	); err != nil {
		return fmt.Errorf("failed to send sign-in verification code email: %w", err)
	}

	type payload struct {
		CodeLength       int    `json:"code_length"`
		CodeRegexPattern string `json:"code_regex_pattern"`
	}

	codeLen := len(code)
	return c.JSON(http.StatusOK, APISuccessResponse{
		Data: payload{
			CodeLength:       codeLen,
			CodeRegexPattern: fmt.Sprintf("^[%s]{%d}$", util.AlphaNumCharset, codeLen),
		},
	})
}

func (h *Handler) VerifySignIn(c echo.Context) error {
	var input struct {
		Email            string `json:"email" validate:"required,email"`
		VerificationCode string `json:"verification_code" validate:"required,len=6"`
	}
	if err := bindAndValidate(c, &input); err != nil {
		return err
	}

	input.Email = strings.ToLower(input.Email)
	ctx := c.Request().Context()
	key := redisPrefixSignInVerificationCode + input.Email
	vals, err := h.Redis.HGetAll(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("failed to exec HGET on redis: %w", err)
	}
	if len(vals) == 0 {
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid verification code")
	}

	hash := vals["hash"]
	attempts, err := strconv.Atoi(vals["attempts"])
	if err != nil {
		return fmt.Errorf("failed to convert 'attempts' to type int: %w", err)
	}

	cfg := h.Config.Get()
	if attempts >= cfg.MaxVerificationAttempts {
		if err = h.Redis.Del(ctx, key).Err(); err != nil {
			return fmt.Errorf("failed to delete verification code: %w", err)
		}
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid verification code")
	}

	if !util.VerifySecureHash([]byte(input.VerificationCode), []byte(hash)) {
		if err = h.Redis.HIncrBy(ctx, key, "attempts", 1).Err(); err != nil {
			return fmt.Errorf("failed to increment verification code attempts: %w", err)
		}
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid verification code")
	}

	if err = h.Redis.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete verification code: %w", err)
	}

	user, err := h.Repo.UpsertUser(c.Request().Context(), input.Email)
	if err != nil {
		return fmt.Errorf("failed to upsert user: %w", err)
	}

	sessionId, err := util.GenerateSecret("session", 32, util.EncodingBase64URL)
	if err != nil {
		return fmt.Errorf("failed to generate session id: %w", err)
	}

	if err = h.Redis.Set(ctx, "session:"+sessionId, user.ID, cfg.SessionTTL).Err(); err != nil {
		return fmt.Errorf("failed to save session to redis: %w", err)
	}

	sess, err := session.Get("session", c)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	sess.Values["id"] = sessionId

	if err = sess.Save(c.Request(), c.Response()); err != nil {
		return fmt.Errorf("failed to save session: %w", err)
	}

	return c.NoContent(http.StatusOK)
}

func (h *Handler) SignOut(c echo.Context) error {
	sess, err := session.Get("session", c)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "user is not signed in")
	}

	sess.Options.MaxAge = -1

	if err = sess.Save(c.Request(), c.Response()); err != nil {
		return fmt.Errorf("failed to save session: %w", err)
	}

	return c.NoContent(http.StatusOK)
}
