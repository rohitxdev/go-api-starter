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
	"time"

	"github.com/rohitxdev/go-api-starter/internal/blobstore"
	"github.com/rohitxdev/go-api-starter/internal/config"
	"github.com/rohitxdev/go-api-starter/internal/database"
	"github.com/rohitxdev/go-api-starter/internal/email"
	"github.com/rohitxdev/go-api-starter/internal/kv"
	"github.com/rohitxdev/go-api-starter/internal/prettylog"
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

	//Load config
	c, err := config.Load()
	if err != nil {
		panic("load config: " + err.Error())
	}

	//Set up logger
	logOpts := slog.HandlerOptions{
		Level: slog.LevelDebug,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Value.String() == "" || a.Value.Equal(slog.AnyValue(nil)) {
				return slog.Attr{}
			}
			return a
		},
	}

	var logHandler slog.Handler
	if c.AppEnv == config.EnvDevelopment {
		logHandler = prettylog.NewHandler(os.Stderr, &logOpts)
	} else {
		logHandler = slog.NewJSONHandler(os.Stderr, &logOpts)
	}

	slog.SetDefault(slog.New(logHandler))

	slog.Debug(fmt.Sprintf("running %s on %s in %s environment", config.BuildId, runtime.GOOS+"/"+runtime.GOARCH, c.AppEnv))

	// Set maxprocs logger
	maxprocsLogger := maxprocs.Logger(func(s string, i ...interface{}) {
		slog.Debug(fmt.Sprintf(s, i...))
	})

	if _, err = maxprocs.Set(maxprocsLogger); err != nil {
		panic("set maxprocs logger: " + err.Error())
	}

	//Connect to postgres database
	db, err := database.NewPostgres(c.DatabaseUrl)
	if err != nil {
		panic("connect to database: " + err.Error())
	}
	defer func() {
		if err = db.Close(); err != nil {
			panic("close database: " + err.Error())
		}
		slog.Debug("database connection closed")
	}()
	slog.Debug("connected to database")

	//Connect to sqlite database
	sqliteDb, err := database.NewSqlite(":memory:")
	if err != nil {
		panic("connect to sqlite database: " + err.Error())
	}

	//Connect to kv store
	kv, err := kv.New(sqliteDb, time.Minute*5)
	if err != nil {
		panic("connect to KV store: " + err.Error())
	}
	defer func() {
		kv.Close()
		slog.Debug("kv store closed")
	}()
	slog.Debug("connected to kv store")

	//Create API handler
	r, err := repo.New(db)
	if err != nil {
		panic("create repo: " + err.Error())
	}
	defer r.Close()

	s3Client, err := blobstore.New(c.S3Endpoint, c.S3DefaultRegion, c.AwsAccessKeyId, c.AwsAccessKeySecret)
	if err != nil {
		panic("connect to s3 client: " + err.Error())
	}

	h := routes.NewHandler(&routes.Dependencies{
		Config:     c,
		KVStore:    kv,
		Repo:       r,
		Email:      &email.Client{},
		BlobStore:  s3Client,
		FileSystem: &fileSystem,
	})
	if err != nil {
		panic("create handler: " + err.Error())
	}
	e, err := routes.NewRouter(h)
	if err != nil {
		panic("create router: " + err.Error())
	}

	//Create tcp listener & start server
	ls, err := net.Listen("tcp", c.Address)
	if err != nil {
		panic("tcp listen: " + err.Error())
	}
	defer func() {
		if err = ls.Close(); err != nil {
			panic("close tcp listener: " + err.Error())
		}
		slog.Debug("tcp listener closed")
	}()
	slog.Debug("tcp listener created")

	go func() {
		if err := http.Serve(ls, e); err != nil && !errors.Is(err, net.ErrClosed) {
			panic("serve http: " + err.Error())
		}
	}()

	slog.Debug("http server started")
	slog.Info(fmt.Sprintf("server is listening to http://%s and is ready to serve requests", ls.Addr()))

	//Shut down http server gracefully
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	<-ctx.Done()

	ctx, cancel = context.WithTimeout(context.Background(), c.ShutdownTimeout)
	defer cancel()

	if err := e.Shutdown(ctx); err != nil {
		panic("http server shutdown: " + err.Error())
	}

	slog.Debug("http server shut down gracefully")
}
