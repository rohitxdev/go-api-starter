//go:generate go run github.com/swaggo/swag/cmd/swag@latest init -q -g internal/routes/router.go
package main

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"log/slog"
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
	"github.com/rohitxdev/go-api-starter/internal/kv"
	"github.com/rohitxdev/go-api-starter/internal/logger"
	"github.com/rohitxdev/go-api-starter/internal/repo"
	"github.com/rohitxdev/go-api-starter/internal/routes"
	"go.uber.org/automaxprocs/maxprocs"
)

//go:embed web docs
var fileSystem embed.FS

func main() {
	if config.BuildId == "" {
		panic("build id is not set")
	}

	if _, err := maxprocs.Set(); err != nil {
		panic("could not set maxprocs: " + err.Error())
	}

	//Load config
	c, err := config.Load()
	if err != nil {
		panic("could not load config: " + err.Error())
	}

	//Set up logger
	loggerOpts := logger.HandlerOptions{
		Level: slog.LevelDebug,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Value.String() == "" || a.Value.Equal(slog.AnyValue(nil)) {
				return slog.Attr{}
			}
			return a
		},
		NoColor: !c.IsDev,
	}
	slog.SetDefault(logger.New(os.Stderr, &loggerOpts))

	slog.Debug(fmt.Sprintf("BuildId: %s, Platform: %s/%s, MaxProcs: %d, Env: %s", config.BuildId, runtime.GOOS, runtime.GOARCH, runtime.GOMAXPROCS(0), c.Env))

	//Connect to postgres database
	db, err := database.NewPostgres(c.DatabaseUrl)
	if err != nil {
		panic("could not connect to database: " + err.Error())
	}
	defer func() {
		if err = db.Close(); err != nil {
			panic("could not close database: " + err.Error())
		}
		slog.Debug("Database connection closed")
	}()
	slog.Debug("Connected to database")

	//Connect to sqlite database
	sqliteDb, err := database.NewSqlite(":memory:")
	if err != nil {
		panic("could not connect to sqlite database: " + err.Error())
	}

	//Connect to kv store
	kv, err := kv.New(sqliteDb, time.Minute*5)
	if err != nil {
		panic("could not connect to kv store: " + err.Error())
	}
	defer func() {
		kv.Close()
		slog.Debug("KV store closed")
	}()
	slog.Debug("Connected to KV store")

	//Create API handler
	r, err := repo.New(db)
	if err != nil {
		panic("could not create repo: " + err.Error())
	}
	defer r.Close()

	s3Client, err := blobstore.New(c.S3Endpoint, c.S3DefaultRegion, c.AwsAccessKeyId, c.AwsAccessKeySecret)
	if err != nil {
		panic("could not connect to s3 client: " + err.Error())
	}

	emailTemplates, err := template.ParseFS(fileSystem, "web/templates/emails/*.tmpl")
	if err != nil {
		panic("could not parse email templates: " + err.Error())
	}
	emailClient := email.New(&email.SmtpCredentials{
		Host:     c.SmtpHost,
		Port:     c.SmtpPort,
		Username: c.SmtpUsername,
		Password: c.SmtpPassword,
	}, emailTemplates)

	h := routes.NewHandler(&routes.Dependencies{
		Config:     c,
		KVStore:    kv,
		Repo:       r,
		Email:      emailClient,
		BlobStore:  s3Client,
		FileSystem: &fileSystem,
	})
	if err != nil {
		panic("could not create handler: " + err.Error())
	}
	e, err := routes.NewRouter(h)
	if err != nil {
		panic("could not create router: " + err.Error())
	}

	//Start HTTP server
	ls, err := net.Listen("tcp", c.Address)
	if err != nil {
		panic("could not listen on tcp: " + err.Error())
	}
	defer func() {
		if err = ls.Close(); err != nil {
			panic("could not close tcp listener: " + err.Error())
		}
	}()

	go func() {
		if err := http.Serve(ls, e); err != nil && !errors.Is(err, net.ErrClosed) {
			panic("could not serve http: " + err.Error())
		}
	}()

	slog.Debug("HTTP server started")
	slog.Info(fmt.Sprintf("Server is listening on http://%s and is ready to serve requests", ls.Addr()))

	//Shut down HTTP server gracefully
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	<-ctx.Done()

	ctx, cancel = context.WithTimeout(context.Background(), c.ShutdownTimeout)
	defer cancel()

	if err := e.Shutdown(ctx); err != nil {
		panic("could not shutdown http server: " + err.Error())
	}

	slog.Debug("HTTP server shut down gracefully")
}
