//go:generate go run github.com/swaggo/swag/cmd/swag@latest init -q -g internal/routes/router.go
package main

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"text/template"
	"time"

	"github.com/rohitxdev/go-api-starter/internal/blobstore"
	"github.com/rohitxdev/go-api-starter/internal/config"
	"github.com/rohitxdev/go-api-starter/internal/database"
	"github.com/rohitxdev/go-api-starter/internal/email"
	"github.com/rohitxdev/go-api-starter/internal/kvstore"
	"github.com/rohitxdev/go-api-starter/internal/logger"
	"github.com/rohitxdev/go-api-starter/internal/repo"
	"github.com/rohitxdev/go-api-starter/internal/routes"
	"go.uber.org/automaxprocs/maxprocs"
)

//go:embed public templates docs
var fs embed.FS

func main() {
	if _, err := maxprocs.Set(); err != nil {
		panic("Failed to set maxprocs: " + err.Error())
	}

	//Load config
	cfg, err := config.Load()
	if err != nil {
		panic("Failed to load config: " + err.Error())
	}

	//Set up logger
	logr := logger.New(os.Stderr, cfg.IsDev)

	logr.Debug().
		Str("AppVersion", cfg.AppVersion).
		Str("BuildType", cfg.BuildType).
		Str("Env", cfg.Env).
		Int("MaxProcs", runtime.GOMAXPROCS(0)).
		Str("Platform", fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)).
		Msg("Running " + cfg.AppName)

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

	emailTemplates, err := template.ParseFS(fs, "templates/emails/*.tmpl")
	if err != nil {
		panic("Failed to parse email templates: " + err.Error())
	}
	emailer := email.New(&email.SMTPCredentials{
		Host:     cfg.SMTPHost,
		Port:     cfg.SMTPPort,
		Username: cfg.SMTPUsername,
		Password: cfg.SMTPPassword,
	}, emailTemplates)

	e, err := routes.NewRouter(&routes.Services{
		BlobStore:  bs,
		Config:     cfg,
		EmbeddedFS: &fs,
		Email:      emailer,
		KVStore:    kv,
		Logger:     logr,
		Repo:       r,
	})
	if err != nil {
		panic("Failed to create router: " + err.Error())
	}

	ls, err := net.Listen("tcp", net.JoinHostPort(cfg.Host, cfg.Port))
	if err != nil {
		panic("Failed to listen on TCP: " + err.Error())
	}
	defer func() {
		if err = ls.Close(); err != nil {
			panic("Failed to close TCP listener: " + err.Error())
		}
	}()

	//Start HTTP server
	go func() {
		if err := http.Serve(ls, e); err != nil && !errors.Is(err, net.ErrClosed) {
			panic("Failed to serve HTTP: " + err.Error())
		}
	}()

	logr.Debug().Msg("HTTP server started")
	logr.Info().Msg(fmt.Sprintf("Server is listening on http://%s and is ready to serve requests", ls.Addr()))

	//Shut down HTTP server gracefully
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	<-ctx.Done()

	ctx, cancel = context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := e.Shutdown(ctx); err != nil {
		panic("Failed to shutdown http server: " + err.Error())
	}

	logr.Debug().Msg("HTTP server shut down gracefully")
}
