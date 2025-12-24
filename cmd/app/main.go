package main

import (
	"context"
	"fmt"
	"html/template"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path"
	"runtime"
	"syscall"
	"time"

	"github.com/rohitxdev/go-api/assets"
	"github.com/rohitxdev/go-api/database/repository"
	"github.com/rohitxdev/go-api/deps/cache"
	"github.com/rohitxdev/go-api/deps/config"
	"github.com/rohitxdev/go-api/deps/email"
	"github.com/rohitxdev/go-api/deps/postgres"
	"github.com/rohitxdev/go-api/deps/redis"
	"github.com/rohitxdev/go-api/handler"
)

func run() error {
	ctx := context.Background()

	signals := []os.Signal{os.Interrupt, syscall.SIGTERM}
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, signals...)

	// Logger
	var level slog.LevelVar
	slogOpts := slog.HandlerOptions{
		Level: &level,
	}
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slogOpts))
	slog.SetDefault(logger)

	// Config
	configStore, err := config.NewStore()
	if err != nil {
		return fmt.Errorf("failed to create config store: %w", err)
	}

	cfg := configStore.Get()
	if cfg.Debug {
		level.Set(slog.LevelDebug)
	}

	// Email
	templates, err := template.ParseFS(assets.FS, "templates/emails/*.tmpl")
	if err != nil {
		return fmt.Errorf("failed to parse email templates: %w", err)
	}
	ec, err := email.New(&email.SMTPCredentials{}, templates)
	if err != nil {
		return fmt.Errorf("failed to initialize email client: %w", err)
	}

	// Cache
	cache, err := cache.New[string](time.Hour * 12)
	if err != nil {
		return fmt.Errorf("failed to initialize cache: %w", err)
	}
	logger.Info("initialized cache")

	// Postgres
	pg, err := postgres.New(ctx, cfg.PostgresURL)
	if err != nil {
		return fmt.Errorf("failed to connect to postgres server: %w", err)
	}
	defer pg.Close()
	logger.Info("connected to postgres server")
	repo := repository.New(pg)

	// Redis
	rdb, err := redis.New(ctx, cfg.RedisURL, cfg.AppName)
	if err != nil {
		return fmt.Errorf("failed to connect to redis server: %w", err)
	}
	defer rdb.Close()
	logger.Info("connected to redis server")

	deps := handler.Dependencies{
		Config: configStore,
		Cache:  cache,
		Redis:  rdb,
		Repo:   repo,
		Logger: logger,
		Email:  ec,
	}

	listener, err := net.Listen("tcp", net.JoinHostPort(cfg.HTTPHost, cfg.HTTPPort))
	if err != nil {
		return fmt.Errorf("failed to acquire TCP listener: %w", err)
	}

	h, err := handler.New(&deps)
	if err != nil {
		return fmt.Errorf("failed to create http handler: %w", err)
	}

	server := &http.Server{
		Handler:      h,
		ReadTimeout:  time.Minute,
		WriteTimeout: time.Minute,
		IdleTimeout:  time.Minute,
		ConnState: func(c net.Conn, configStore http.ConnState) {
			logger.Debug("HTTP connection state changed", "client_ip", c.RemoteAddr().String(), "state", configStore.String())
		},
	}

	errCh := make(chan error)
	go func() {
		var serveErr error

		if cfg.AppEnv == config.EnvProduction {
			serveErr = server.Serve(listener)
		} else {
			certPath := path.Join(cfg.TmpDir, "localhost.crt")
			keyPath := path.Join(cfg.TmpDir, "localhost.key")
			serveErr = server.ServeTLS(listener, certPath, keyPath)
		}

		if serveErr != nil && serveErr != http.ErrServerClosed {
			errCh <- fmt.Errorf("HTTP server error: %w", serveErr)
		}

		close(errCh)
	}()

	logger.Info("application is running",
		slog.Group("build", slog.String("type", cfg.BuildType), slog.Time("timestamp", cfg.BuildTimestamp), slog.String("version", cfg.AppVersion)),
		slog.Group("http", slog.String("address", listener.Addr().String())),
		slog.Group("runtime", slog.String("platform", fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH))), slog.String("environment", cfg.AppEnv),
	)

	select {
	case s := <-sigCh:
		logger.Info("received shutdown signal, attempting to shutdown gracefully", slog.String("signal", s.String()))

		go func() {
			s := <-sigCh
			logger.Warn("received second shutdown signal, forcing exit", slog.String("signal", s.String()))
			os.Exit(1)
		}()

		shutdownCtx, cancel := context.WithTimeout(ctx, time.Minute)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("failed to shut down HTTP server gracefully: %w", err)
		}
		logger.Info("HTTP server shut down gracefully")

		return nil
	case err := <-errCh:
		return err
	}
}

func main() {
	if err := run(); err != nil {
		slog.Error("application terminated", "error", err)
	}
}
