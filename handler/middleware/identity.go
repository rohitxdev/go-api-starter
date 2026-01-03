package middleware

import (
	"github.com/google/uuid"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
	"github.com/rohitxdev/go-api/deps/config"
	"github.com/rohitxdev/go-api/util"
)

const (
	HeaderXDeviceID          = "X-Device-ID"
	HeaderXDeviceIDSignature = "X-Device-ID-Signature"
)

const (
	IDKeyPrefixUser   = "user"
	IDKeyPrefixDevice = "device"
	IDKeyPrefixIP     = "ip"
)

const (
	CtxKeyIDKey  = "id_key"
	CtxKeyUserID = "user_id"
	CtxKeyUser   = "user"
)

func resolveUserFromSession(c echo.Context, rdb *redis.Client) string {
	sess, err := session.Get("session", c)
	if err != nil {
		return ""
	}

	sessIDstr, ok := sess.Values["id"].(string)
	if !ok {
		return ""
	}

	sessID, err := uuid.Parse(sessIDstr)
	if err != nil {
		return ""
	}

	userID := rdb.Get(c.Request().Context(), "session:"+sessID.String()).String()
	if userID == "" {
		return ""
	}

	return userID
}

func resolveDevice(c echo.Context, secret string) string {
	header := c.Request().Header
	id := header.Get(HeaderXDeviceID)
	sig := header.Get(HeaderXDeviceIDSignature)

	if id == "" || sig == "" || !util.VerifyHMAC(id, sig, secret) {
		return ""
	}

	return id
}

func ResolveIdentityKey(rdb *redis.Client, cfg *config.Store) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			var prefix, id string

			if userID := resolveUserFromSession(c, rdb); userID != "" {
				prefix = IDKeyPrefixUser
				id = userID
				c.Set(CtxKeyUserID, userID)

			} else if deviceId := resolveDevice(c, cfg.Get().DeviceIDSecret); deviceId != "" {
				prefix = IDKeyPrefixDevice
				id = deviceId

			} else {
				prefix = IDKeyPrefixIP
				id = c.RealIP()
			}

			c.Set(CtxKeyIDKey, prefix+":"+id)
			return next(c)
		}
	}
}
