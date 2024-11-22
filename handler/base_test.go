package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/rohitxdev/go-api-starter/blobstore"
	"github.com/rohitxdev/go-api-starter/config"
	"github.com/rohitxdev/go-api-starter/database"
	"github.com/rohitxdev/go-api-starter/email"
	"github.com/rohitxdev/go-api-starter/handler"
	"github.com/rohitxdev/go-api-starter/kvstore"
	"github.com/rohitxdev/go-api-starter/logger"
	"github.com/rohitxdev/go-api-starter/repo"
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
	//Load config
	cfg, err := config.Load()
	t.Log()
	if err != nil {
		panic("Failed to load config: " + err.Error())
	}

	//Set up logger
	logr := logger.New(os.Stderr, cfg.IsDev)
	//Connect to postgres database
	db, err := database.NewPostgreSQL(cfg.DatabaseURL)
	if err != nil {
		panic("Failed to connect to database: " + err.Error())
	}
	logr.Debug().Msg("Connected to database")
	defer func() {
		if err = db.Close(); err != nil {
			panic("Failed to close database: " + err.Error())
		}
		logr.Debug().Msg("Database connection closed")
	}()

	//Connect to KV store
	kv, err := kvstore.New("kv", time.Minute*5)
	if err != nil {
		panic("Failed to connect to KV store: " + err.Error())
	}

	logr.Debug().Msg("Connected to KV store")
	defer func() {
		kv.Close()
		logr.Debug().Msg("KV store closed")
	}()

	// Create repo
	r, err := repo.New(db)
	if err != nil {
		panic("Failed to create repo: " + err.Error())
	}
	defer r.Close()

	bs, err := blobstore.New(cfg.S3Endpoint, cfg.S3DefaultRegion, cfg.AWSAccessKeyID, cfg.AWSAccessKeySecret)
	if err != nil {
		panic("Failed to connect to S3 client: " + err.Error())
	}
	e, err := email.New(&email.SMTPCredentials{})
	assert.Nil(t, err)

	h, err := handler.New(&handler.Services{
		BlobStore: bs,
		Config:    cfg,
		KVStore:   kv,
		Logger:    logr,
		Repo:      r,
		Email:     e,
	})
	assert.Nil(t, err)

	t.Run("GET /", func(t *testing.T) {
		req, err := createHttpRequest(&httpRequestOpts{
			method: http.MethodGet,
			path:   "/",
		})
		assert.Nil(t, err)
		res := httptest.NewRecorder()
		h.ServeHTTP(res, req)
		assert.Equal(t, http.StatusOK, res.Code)
	})

	t.Run("GET /ping", func(t *testing.T) {
		req, err := createHttpRequest(&httpRequestOpts{
			method: http.MethodGet,
			path:   "/ping",
		})
		assert.Nil(t, err)
		res := httptest.NewRecorder()
		h.ServeHTTP(res, req)
		assert.Equal(t, http.StatusOK, res.Code)
	})

	t.Run("GET /config", func(t *testing.T) {
		req, err := createHttpRequest(&httpRequestOpts{
			method: http.MethodGet,
			path:   "/config",
		})
		assert.Nil(t, err)
		res := httptest.NewRecorder()
		h.ServeHTTP(res, req)
		assert.Equal(t, http.StatusOK, res.Code)
	})
}
