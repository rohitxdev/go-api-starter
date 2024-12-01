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
	"github.com/rohitxdev/go-api-starter/cryptoutil"
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
	// Set GOMAXPROCS to match the Linux container CPU quota.
	if _, err := maxprocs.Set(); err != nil {
		return fmt.Errorf("failed to set maxprocs: %w", err)
	}

	// Load config.
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Set up logger.
	logr := logger.New(os.Stderr, cfg.IsDev)

	logr.Debug().
		Str("appVersion", cfg.AppVersion).
		Str("buildType", cfg.BuildType).
		Str("env", cfg.Env).
		Int("maxProcs", runtime.GOMAXPROCS(0)).
		Str("platform", fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)).
		Msg("Running " + cfg.AppName)

	// Connect to KV store for caching.
	kv, err := kvstore.New("kv", time.Minute*10)
	if err != nil {
		return fmt.Errorf("failed to connect to KV store: %w", err)
	}

	// Connect to postgres database.
	db, err := database.NewPostgreSQL(cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// Create repo for interacting with the database.
	r, err := repo.New(db)
	if err != nil {
		return fmt.Errorf("failed to create repo: %w", err)
	}

	// Create blobstore for storing files.
	bs, err := blobstore.New(cfg.S3Endpoint, cfg.S3DefaultRegion, cfg.AWSAccessKeyID, cfg.AWSAccessKeySecret)
	if err != nil {
		return fmt.Errorf("failed to connect to S3 client: %w", err)
	}

	e, err := email.New(&email.SMTPCredentials{
		Host:               cfg.SMTPHost,
		Port:               cfg.SMTPPort,
		Username:           cfg.SMTPUsername,
		Password:           cfg.SMTPPassword,
		InsecureSkipVerify: true,
	})
	if err != nil {
		return fmt.Errorf("failed to create email client: %w", err)
	}

	s := handler.Service{
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
		return fmt.Errorf("failed to create HTTP handler: %w", err)
	}

	errCh := make(chan error)
	address := net.JoinHostPort(cfg.Host, cfg.Port)
	isDevTLS := cfg.IsDev && cfg.UseDevTLS

	// Start HTTP server.
	go func() {
		if isDevTLS {
			// nolint
			certPath, keyPath, err := cryptoutil.GenerateSelfSignedCert()
			if err != nil {
				errCh <- fmt.Errorf("failed to generate self-signed certificate: %w", err)
			}
			errCh <- http.ListenAndServeTLS(address, certPath, keyPath, h)
		} else {
			// Stdlib supports HTTP/2 by default when serving over TLS, but has to be explicitly enabled otherwise.
			h2Handler := h2c.NewHandler(h, &http2.Server{})
			errCh <- http.ListenAndServe(address, h2Handler)
		}
	}()

	proto := "http"
	if isDevTLS {
		proto = "https"
	}
	logr.Info().Msg(fmt.Sprintf("Server is listening on %s://%s", proto, address))

	// Shut down HTTP server gracefully.
	ctx := context.Background()
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt)
	defer cancel()

	select {
	case <-ctx.Done():
		ctx, cancel = context.WithTimeout(ctx, time.Second*10)
		defer cancel()

		if err = h.Shutdown(ctx); err != nil {
			return fmt.Errorf("failed to shutdown HTTP server: %w", err)
		}

		logr.Debug().Msg("HTTP server shut down gracefully")
	case err = <-errCh:
		if err != nil && !errors.Is(err, net.ErrClosed) {
			err = fmt.Errorf("failed to start HTTP server: %w", err)
		}
	}
	return err
}
