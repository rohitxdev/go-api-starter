package routes_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/rohitxdev/go-api-starter/internal/config"
	"github.com/rohitxdev/go-api-starter/internal/routes"
	"github.com/stretchr/testify/assert"
)

type httpRequestOpts struct {
	query   map[string]string
	body    echo.Map
	headers map[string]string
	method  string
	path    string
}

func createHttpRequest(opts *httpRequestOpts) (*http.Request, error) {
	url, err := url.Parse(opts.path)
	if err != nil {
		return nil, err
	}
	q := url.Query()
	for key, value := range opts.query {
		q.Set(key, value)
	}
	url.RawQuery = q.Encode()
	j, err := json.Marshal(opts.body)
	if err != nil {
		return nil, err
	}
	req := httptest.NewRequest(opts.method, url.String(), bytes.NewReader(j))
	for key, value := range opts.headers {
		req.Header.Set(key, value)
	}
	return req, err
}

func TestRootRoutes(t *testing.T) {
	cfg, err := config.Load()
	assert.Nil(t, err)
	h := routes.NewHandler(&routes.Dependencies{
		Config: cfg,
	})
	assert.Nil(t, err)
	e, err := routes.NewRouter(h)
	assert.Nil(t, err)

	t.Run("GET /", func(t *testing.T) {
		req, err := createHttpRequest(&httpRequestOpts{
			method: http.MethodGet,
			path:   "/",
		})
		assert.Nil(t, err)
		res := httptest.NewRecorder()
		c := e.NewContext(req, res)
		_ = h.GetPing(c)
		assert.Equal(t, http.StatusOK, res.Code)
	})

	t.Run("GET /ping", func(t *testing.T) {
		req, err := createHttpRequest(&httpRequestOpts{
			method: http.MethodGet,
			path:   "/ping",
		})
		assert.Nil(t, err)
		res := httptest.NewRecorder()
		c := e.NewContext(req, res)
		_ = h.GetPing(c)
		assert.Equal(t, http.StatusOK, res.Code)
		assert.Equal(t, "pong", res.Body.String())
	})

	t.Run("GET /config", func(t *testing.T) {
		req, err := createHttpRequest(&httpRequestOpts{
			method: http.MethodGet,
			path:   "/config",
		})
		assert.Nil(t, err)
		res := httptest.NewRecorder()
		c := e.NewContext(req, res)
		_ = h.GetConfig(c)
		assert.Equal(t, http.StatusOK, res.Code)
	})
}
