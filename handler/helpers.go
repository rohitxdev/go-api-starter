package handler

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"text/template"

	"github.com/go-playground/validator"
	gojson "github.com/goccy/go-json"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/rohitxdev/go-api/config"
	"github.com/rohitxdev/go-api/database/repository"
)

type APIResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

type APIResponseWithPayload struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Payload any    `json:"payload,omitzero"`
}

var MsgInternalServerError = "something went wrong"

var cfg = config.Config

// Custom view renderer
type viewRenderer struct {
	templates *template.Template
}

func (v viewRenderer) Render(w io.Writer, name string, data any, c echo.Context) error {
	return v.templates.ExecuteTemplate(w, name+".tmpl", data)
}

// Custom request validator
type requestValidator struct {
	validator *validator.Validate
}

func (v requestValidator) Validate(i any) error {
	if err := v.validator.Struct(i); err != nil {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, err)
	}
	return nil
}

// Custom JSON serializer & deserializer
type jsonSerializer struct{}

func (s jsonSerializer) Serialize(c echo.Context, data any, indent string) error {
	enc := gojson.NewEncoder(c.Response())
	enc.SetIndent("", indent)
	return enc.Encode(data)
}

func (s jsonSerializer) Deserialize(c echo.Context, v any) error {
	dec := gojson.NewDecoder(c.Request().Body)
	err := dec.Decode(v)
	if ute, ok := err.(*gojson.UnmarshalTypeError); ok {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Unmarshal type error: expected=%v, got=%v, field=%v, offset=%v", ute.Type, ute.Value, ute.Field, ute.Offset)).SetInternal(err)
	} else if se, ok := err.(*gojson.SyntaxError); ok {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Syntax error: offset=%v, error=%v", se.Offset, se.Error())).SetInternal(err)
	}
	return err
}

// bindAndValidate binds path params, query params and the request body into provided type `i` and validates provided `i`. `i` must be a pointer. The default binder binds body based on Content-Type header. Validator must be registered using `Echo#Validator`.
func bindAndValidate(c echo.Context, i any) error {
	var err error
	if err = c.Bind(i); err != nil {
		return err
	}
	binder := echo.DefaultBinder{}
	if err = binder.BindHeaders(c, i); err != nil {
		return err
	}
	if err = c.Validate(i); err != nil {
		return err
	}
	return err
}

func canonicalizeEmail(email string) string {
	email = strings.TrimSpace(email)
	email = strings.ToLower(email)
	parts := strings.Split(email, "@")
	username := parts[0]
	domain := parts[1]
	if strings.Contains(username, "+") {
		username = strings.Split(username, "+")[0]
	}
	username = strings.ReplaceAll(username, ".", "")
	return username + "@" + domain
}

func getCurrentUser(c echo.Context, repo repository.Querier) *repository.User {
	user, ok := c.Get("user").(*repository.User)
	if ok {
		return user
	}

	sess, err := session.Get("session", c)
	if err != nil {
		return nil
	}

	sesssionIDStr, ok := sess.Values["session_id"].(string)
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

func serverError(err error) *echo.HTTPError {
	return echo.NewHTTPError(http.StatusInternalServerError, err)
}
