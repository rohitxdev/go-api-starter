package app

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"time"

	"github.com/rohitxdev/go-api-starter/blobstore"
	"github.com/rohitxdev/go-api-starter/config"
	"github.com/rohitxdev/go-api-starter/database"
	"github.com/rohitxdev/go-api-starter/email"
	"github.com/rohitxdev/go-api-starter/handler"
	"github.com/rohitxdev/go-api-starter/kvstore"
	"github.com/rohitxdev/go-api-starter/logger"
	"github.com/rohitxdev/go-api-starter/repo"
	"go.uber.org/automaxprocs/maxprocs"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

func Run() error {
	// Set max threads to match the Linux container CPU quota.
	if _, err := maxprocs.Set(); err != nil {
		panic("Failed to set maxprocs: " + err.Error())
	}
	//Load config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("Failed to load config: %w", err)
	}

	//Set up logger
	logr := logger.New(os.Stderr, cfg.IsDev)

	logr.Debug().
		Str("appVersion", cfg.AppVersion).
		Str("buildType", cfg.BuildType).
		Str("env", cfg.Env).
		Int("maxProcs", runtime.GOMAXPROCS(0)).
		Str("platform", fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)).
		Msg("Running " + cfg.AppName)

	//Connect to postgres database
	db, err := database.NewPostgreSQL(cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("Failed to connect to database: %w", err)
	}

	//Connect to KV store
	kv, err := kvstore.New("kv", time.Minute*5)
	if err != nil {
		return fmt.Errorf("Failed to connect to KV store: %w", err)
	}

	// Create repo
	r, err := repo.New(db)
	if err != nil {
		return fmt.Errorf("Failed to create repo: %w", err)
	}

	bs, err := blobstore.New(cfg.S3Endpoint, cfg.S3DefaultRegion, cfg.AWSAccessKeyID, cfg.AWSAccessKeySecret)
	if err != nil {
		return fmt.Errorf("Failed to connect to S3 client: %w", err)
	}

	e, err := email.New(&email.SMTPCredentials{
		Host:     cfg.SMTPHost,
		Port:     cfg.SMTPPort,
		Username: cfg.SMTPUsername,
		Password: cfg.SMTPPassword,
	})
	if err != nil {
		return fmt.Errorf("Failed to create email client: %w", err)
	}

	s := handler.Services{
		BlobStore: bs,
		Config:    cfg,
		Email:     e,
		KVStore:   kv,
		Logger:    logr,
		Repo:      r,
	}
	defer s.Close()

	h, err := handler.New(&s)
	if err != nil {
		return fmt.Errorf("Failed to create router: %w", err)
	}

	ls, err := net.Listen("tcp", net.JoinHostPort(cfg.Host, cfg.Port))
	if err != nil {
		return fmt.Errorf("Failed to listen on TCP: %w", err)
	}
	defer ls.Close()

	//Start HTTP server
	go func() {
		// Stdlib supports HTTP/2 by default when serving over TLS, but has to be explicitly enabled otherwise.
		handler := h2c.NewHandler(h, &http2.Server{})
		if err = http.Serve(ls, handler); err != nil && !errors.Is(err, net.ErrClosed) {
			err = fmt.Errorf("Failed to serve HTTP: %w", err)
		}
	}()

	logr.Debug().Msg("HTTP server started")
	logr.Info().Msg(fmt.Sprintf("Server is listening on http://%s and is ready to serve requests", ls.Addr()))

	ctx := context.Background()
	//Shut down HTTP server gracefully
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt)
	defer cancel()

	<-ctx.Done()

	ctx, cancel = context.WithTimeout(ctx, cfg.ShutdownTimeout)
	defer cancel()

	if err = h.Shutdown(ctx); err != nil {
		return fmt.Errorf("Failed to shutdown HTTP server: %w", err)
	}

	logr.Debug().Msg("HTTP server shut down gracefully")

	return err
}
