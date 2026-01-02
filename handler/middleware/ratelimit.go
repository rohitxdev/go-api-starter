package middleware

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/go-redis/redis_rate/v10"
	"github.com/labstack/echo/v4"
)

var (
	ErrIDKeyNotPresent = errors.New("identity key not present")
)

func RateLimit(limiter *redis_rate.Limiter, rate redis_rate.Limit) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			idKey, ok := c.Get(CtxKeyIDKey).(string)
			if !ok {
				return echo.ErrInternalServerError.SetInternal(ErrIDKeyNotPresent)
			}

			rateLimitKey := "rl:" + idKey
			res, err := limiter.Allow(c.Request().Context(), rateLimitKey, rate)
			if err != nil {
				return echo.ErrInternalServerError.SetInternal(fmt.Errorf("ratelimiter error: %w", err))
			}

			retryAfterSeconds := int(res.RetryAfter.Seconds())
			resetAt := time.Now().Add(res.ResetAfter).Unix()

			c.Response().Header().Set("X-RateLimit-Limit", strconv.Itoa(rate.Rate))
			c.Response().Header().Set("X-RateLimit-Remaining", strconv.Itoa(res.Remaining))
			c.Response().Header().Set("X-RateLimit-Reset", strconv.FormatInt(resetAt, 10))

			if res.Allowed == 0 {
				if retryAfterSeconds > 0 {
					c.Response().Header().Set("Retry-After", strconv.Itoa(retryAfterSeconds))
				}
				return echo.ErrTooManyRequests
			}

			return next(c)
		}
	}
}
