package application

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path"
	"runtime"
	"syscall"
	"time"

	"github.com/rohitxdev/go-api/config"
	"github.com/rohitxdev/go-api/database"
	"github.com/rohitxdev/go-api/database/repository"
	"github.com/rohitxdev/go-api/handler"
)

var cfg = config.Config

func Run() error {
	signals := []os.Signal{os.Interrupt, syscall.SIGTERM}
	ctx, cancel := signal.NotifyContext(context.Background(), signals...)
	defer func() {
		cancel()
		signal.Reset(signals...)
	}()

	slog.SetDefault(NewLogger(cfg.Debug))

	cache, err := NewCache[string](time.Hour)
	if err != nil {
		return fmt.Errorf("failed to initialize cache: %w", err)
	}

	db, err := database.NewPostgres(ctx, cfg.PostgresURL)
	if err != nil {
		return fmt.Errorf("failed to connect to postgres database: %w", err)
	}
	defer db.Close()
	slog.Info("connected to postgres database")

	svc := handler.Services{
		Cache: cache,
		Repo:  repository.New(db),
	}

	listener, err := net.Listen("tcp", net.JoinHostPort(cfg.HTTPHost, cfg.HTTPPort))
	if err != nil {
		return fmt.Errorf("failed to acquire TCP listener: %w", err)
	}

	h, err := handler.New(&svc)
	if err != nil {
		return fmt.Errorf("failed to create http handler: %w", err)
	}

	server := &http.Server{
		Handler: h,
		ConnState: func(c net.Conn, cs http.ConnState) {
			slog.Debug("HTTP connection state changed", "remote_address", c.RemoteAddr().String(), "state", cs.String())
		},
		ReadTimeout: time.Minute,
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

	slog.Info("application is running",
		slog.Group("app", slog.String("name", cfg.AppName), slog.String("version", cfg.AppVersion), slog.String("environment", cfg.AppEnv)),
		slog.Group("build", slog.String("type", cfg.BuildType), slog.Time("timestamp", cfg.BuildTimestamp)),
		slog.Group("http", slog.String("address", listener.Addr().String())),
		slog.Group("runtime", slog.String("platform", fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH))),
	)

	select {
	case <-ctx.Done():
		slog.Info("attempting to shut down gracefully")

		ctx, cancel = context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		if err = server.Shutdown(ctx); err != nil {
			return fmt.Errorf("failed to shut down HTTP server gracefully: %w", err)
		}

		slog.Info("application was shut down gracefully")
		return nil
	case err := <-errCh:
		return err
	}
}
