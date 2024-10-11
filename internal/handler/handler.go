package handler

import (
	"embed"
	"errors"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/rohitxdev/go-api-starter/internal/config"
	"github.com/rohitxdev/go-api-starter/pkg/blobstore"
	"github.com/rohitxdev/go-api-starter/pkg/email"
	"github.com/rohitxdev/go-api-starter/pkg/kvstore"
	"github.com/rohitxdev/go-api-starter/pkg/repo"
)

type handlerOpts struct {
	config     *config.Server
	kvStore    *kvstore.KVStore
	repo       *repo.Repo
	email      *email.Client
	blobstore  *blobstore.Store
	fileSystem *embed.FS
}

func WithConfig(config *config.Server) func(*handlerOpts) {
	return func(ho *handlerOpts) {
		ho.config = config
	}
}

func WithKVStore(kvStore *kvstore.KVStore) func(*handlerOpts) {
	return func(ho *handlerOpts) {
		ho.kvStore = kvStore
	}
}

func WithRepo(repo *repo.Repo) func(*handlerOpts) {
	return func(ho *handlerOpts) {
		ho.repo = repo
	}
}

func WithEmail(email *email.Client) func(*handlerOpts) {
	return func(ho *handlerOpts) {
		ho.email = email
	}
}

func WithBlobStore(fs *blobstore.Store) func(*handlerOpts) {
	return func(ho *handlerOpts) {
		ho.blobstore = fs
	}
}

func WithFileSystem(fileSystem *embed.FS) func(*handlerOpts) {
	return func(ho *handlerOpts) {
		ho.fileSystem = fileSystem
	}
}

type handler handlerOpts

func NewHandler(optFuncs ...func(*handlerOpts)) (*handler, error) {
	opts := handlerOpts{}
	for _, optFunc := range optFuncs {
		optFunc(&opts)
	}

	var errList []error

	if opts.config == nil {
		errList = append(errList, errors.New("config is nil"))
	}
	if opts.kvStore == nil {
		errList = append(errList, errors.New("kvStore is nil"))
	}
	if opts.repo == nil {
		errList = append(errList, errors.New("repo is nil"))
	}
	if opts.email == nil {
		errList = append(errList, errors.New("email is nil"))
	}
	if opts.blobstore == nil {
		errList = append(errList, errors.New("fs is nil"))
	}
	if opts.fileSystem == nil {
		errList = append(errList, errors.New("fileSystem is nil"))
	}

	if len(errList) > 0 {
		return nil, errors.Join(errList...)
	}

	return &handler{
		config:     opts.config,
		kvStore:    opts.kvStore,
		repo:       opts.repo,
		email:      opts.email,
		blobstore:  opts.blobstore,
		fileSystem: opts.fileSystem,
	}, nil
}

// bindAndValidate binds path params, query params and the request body into provided type `i` and validates provided `i`. The default binder binds body based on Content-Type header. Validator must be registered using `Echo#Validator`.
func bindAndValidate(c echo.Context, i any) error {
	var err error
	if err = c.Bind(i); err != nil {
		_ = c.String(http.StatusInternalServerError, err.Error())
		return err
	}
	binder := echo.DefaultBinder{}
	if err = binder.BindHeaders(c, i); err != nil {
		_ = c.String(http.StatusInternalServerError, err.Error())
		return err
	}
	if err = c.Validate(i); err != nil {
		_ = c.String(http.StatusUnprocessableEntity, err.Error())
		return err
	}
	return err
}

func sanitizeEmail(email string) string {
	emailParts := strings.Split(email, "@")
	username := emailParts[0]
	domain := emailParts[1]
	if strings.Contains(username, "+") {
		username = strings.Split(username, "+")[0]
	}
	username = strings.ReplaceAll(username, "-", "")
	username = strings.ReplaceAll(username, ".", "")
	return username + "@" + domain
}

func accepts(c echo.Context) string {
	acceptedTypes := strings.Split(c.Request().Header.Get("Accept"), ",")
	return acceptedTypes[0]
}

// @Summary Ping
// @Description Ping the server.
// @Router /ping [get]
// @Success 200 {string} string "pong"
func (h *handler) Ping(c echo.Context) error {
	return c.String(http.StatusOK, "pong")
}

// @Summary Admin route
// @Description Admin route.
// @Security ApiKeyAuth
// @Router /_ [get]
// @Success 200 {string} string "Hello, Admin!"
// @Failure 401 {string} string "invalid session"
func (h *handler) Admin(c echo.Context) error {
	return c.String(http.StatusOK, "Hello, Admin!")
}

// @Summary Get config
// @Description Get client config.
// @Router /config [get]
// @Success 200 {object} config.Client
func (h *handler) GetConfig(c echo.Context) error {
	clientConfig := config.Client{
		AppEnv: h.config.AppEnv,
	}
	return c.JSON(http.StatusOK, clientConfig)
}
