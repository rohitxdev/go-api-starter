package handlerutil

import (
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v4"
	"github.com/rohitxdev/go-api/database/repository"
)

/*
1. UserId
2. DeviceId
3. IpAddress
*/

const (
	CtxKeyUserID = "user_id"
	CtxKeyUser   = "user"
)

func CurrentUser(c echo.Context, repo repository.Querier) *repository.User {
	user, ok := c.Get(CtxKeyUser).(*repository.User)
	if ok {
		return user
	}

	userID, ok := c.Get(CtxKeyUserID).(string)
	if !ok {
		return nil
	}

	var id pgtype.UUID
	if err := id.Scan(userID); err != nil {
		return nil
	}

	user, err := repo.GetUserByID(c.Request().Context(), id)
	if err != nil {
		return nil
	}

	c.Set(CtxKeyUser, user)

	return user
}
