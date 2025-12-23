package handler

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"text/template"

	"github.com/bytedance/sonic"
	"github.com/bytedance/sonic/decoder"
	"github.com/go-playground/validator"
	"github.com/labstack/echo/v4"
)

type APISuccessResponse struct {
	Data any `json:"data"`
	Meta any `json:"meta,omitempty"`
}

type APIErrorResponse struct {
	Error string `json:"error"`
}

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
	return v.validator.Struct(i)
}

// Custom JSON serializer & deserializer
type JSONSerializer struct{}

func (s JSONSerializer) Serialize(c echo.Context, data any, indent string) error {
	var (
		buf []byte
		err error
	)

	if indent != "" {
		buf, err = sonic.ConfigFastest.MarshalIndent(data, "", indent)
	} else {
		buf, err = sonic.ConfigFastest.Marshal(data)
	}

	if err != nil {
		return fmt.Errorf("failed to serialize JSON object: %w", err)
	}

	_, err = c.Response().Write(buf)
	return err
}

func (s JSONSerializer) Deserialize(c echo.Context, v any) error {
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return err
	}

	if err := sonic.ConfigFastest.Unmarshal(body, v); err != nil {
		if _, ok := err.(*decoder.MismatchTypeError); ok {
			return nil
		}
		if syntaxErr, ok := err.(decoder.SyntaxError); ok {
			return errors.New(syntaxErr.Description())
		}
		return err
	}

	return nil
}

// bindAndValidate binds path params, query params and the request body into provided type `i` and validates provided `i`. `i` must be a pointer. The default binder binds body based on Content-Type header. Validator must be registered using `Echo#Validator`.
func bindAndValidate(c echo.Context, i any) error {
	if err := c.Bind(i); err != nil {
		return err
	}

	binder := echo.DefaultBinder{}
	if err := binder.BindHeaders(c, i); err != nil {
		return err
	}

	if err := c.Validate(i); err != nil {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, err)
	}

	return nil
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
