package handlerutil

import (
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/rohitxdev/go-api/database/repository"
)

func CurrentUser(c echo.Context, repo repository.Querier) *repository.User {
	user, ok := c.Get("user").(*repository.User)
	if ok {
		return user
	}

	sess, err := session.Get("session", c)
	if err != nil {
		return nil
	}

	sesssionIDStr, ok := sess.Values["sessionID"].(string)
	if !ok {
		return nil
	}

	sessionID, err := uuid.Parse(sesssionIDStr)
	if err != nil {
		return nil
	}

	user, err = repo.GetUserBySessionId(c.Request().Context(), pgtype.UUID{Bytes: sessionID, Valid: true})
	if err != nil {
		return nil
	}

	c.Set("user", user)

	return user
}
