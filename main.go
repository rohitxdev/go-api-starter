package main

import (
	"context"
	"embed"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"time"

	"github.com/rohitxdev/go-api-starter/internal/config"
	"github.com/rohitxdev/go-api-starter/internal/handler"
	"github.com/rohitxdev/go-api-starter/pkg/blobstore"
	"github.com/rohitxdev/go-api-starter/pkg/database"
	"github.com/rohitxdev/go-api-starter/pkg/email"
	"github.com/rohitxdev/go-api-starter/pkg/kvstore"
	"github.com/rohitxdev/go-api-starter/pkg/prettylog"
	"github.com/rohitxdev/go-api-starter/pkg/repo"
	_ "go.uber.org/automaxprocs"
)

//go:embed web
var fileSystem embed.FS

func main() {
	//Load config
	envFile := flag.String("env-file", ".env", "Path to .env file")
	flag.Parse()

	c, err := config.Load(*envFile)
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
	if c.Env == config.EnvDevelopment {
		logHandler = prettylog.NewHandler(os.Stderr, &logOpts)
	} else {
		logHandler = slog.NewJSONHandler(os.Stderr, &logOpts)
	}

	slog.SetDefault(slog.New(logHandler))

	slog.Debug(fmt.Sprintf("Running %s on %s in %s environment", c.BuildId, runtime.GOOS+"/"+runtime.GOARCH, c.Env))

	//Connect to postgres database
	db, err := database.NewPostgres(c.DatabaseUrl)
	if err != nil {
		panic("connect to database: " + err.Error())
	}
	defer func() {
		if err = db.Close(); err != nil {
			panic("close database: " + err.Error())
		}
		slog.Debug("Database connection closed ✔︎")
	}()
	slog.Debug("Connected to database ✔︎")

	//Connect to sqlite database
	sqliteDb, err := database.NewSqlite(":memory:")
	if err != nil {
		panic("connect to sqlite database: " + err.Error())
	}

	//Connect to kv store
	kv, err := kvstore.New(sqliteDb, time.Minute*5)
	if err != nil {
		panic("connect to KV store: " + err.Error())
	}
	defer func() {
		kv.Close()
		slog.Debug("KV store closed ✔︎")
	}()
	slog.Debug("Connected to kv store ✔︎")

	//Create API handler
	r := repo.New(db)
	defer r.Close()

	s3Client, err := blobstore.New(c.S3Endpoint, c.S3DefaultRegion, c.AwsAccessKeyId, c.AwsAccessKeySecret)
	if err != nil {
		panic("connect to s3 client: " + err.Error())
	}

	h, err := handler.NewHandler(
		handler.WithConfig(c),
		handler.WithKVStore(kv),
		handler.WithRepo(r),
		handler.WithEmail(&email.Client{}),
		handler.WithBlobStore(s3Client),
		handler.WithFileSystem(&fileSystem),
	)
	if err != nil {
		panic("create handler: " + err.Error())
	}
	e, err := handler.New(h)
	if err != nil {
		panic("create router: " + err.Error())
	}

	//Create tcp listener & start server
	ls, err := net.Listen("tcp", c.Host+":"+c.Port)
	if err != nil {
		panic("tcp listen: " + err.Error())
	}
	defer func() {
		if err = ls.Close(); err != nil {
			panic("close tcp listener: " + err.Error())
		}
		slog.Debug("TCP listener closed ✔︎")
	}()
	slog.Debug("TCP listener created ✔︎")

	go func() {
		if err := http.Serve(ls, e); err != nil && !errors.Is(err, net.ErrClosed) {
			panic("serve http: " + err.Error())
		}
	}()

	slog.Debug("HTTP server started ✔︎")
	slog.Info(fmt.Sprintf("Server is listening to http://%s and is ready to serve requests ✔︎", ls.Addr()))

	//Shut down http server gracefully
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	<-ctx.Done()

	ctx, cancel = context.WithTimeout(context.Background(), c.ShutdownTimeout)
	defer cancel()

	if err := e.Shutdown(ctx); err != nil {
		panic("http server shutdown: " + err.Error())
	}

	slog.Debug("Shut down http server gracefully ✔︎")
}
