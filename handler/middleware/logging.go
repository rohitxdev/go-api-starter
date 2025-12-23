package middleware

import (
	"io"
	"log/slog"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/oklog/ulid/v2"
	"github.com/rohitxdev/go-api/database/repository"
)

const (
	HeaderXClientID = "X-Client-ID"
	HeaderXTraceID  = "X-Trace-ID"
)

type countingReadCloser struct {
	rc io.ReadCloser
	n  int64
}

func (c *countingReadCloser) Read(buf []byte) (int, error) {
	n, err := c.rc.Read(buf)
	c.n += int64(n)
	return n, err
}

func (c *countingReadCloser) Close() error {
	return c.rc.Close()
}

func LogRequest(logger *slog.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			traceID := ulid.Make().String()
			c.Set("traceID", traceID)

			req := c.Request()
			// Hijack request body to count bytes read without copying the body
			var cr *countingReadCloser
			if req.Body != nil {
				cr = &countingReadCloser{rc: req.Body}
				req.Body = cr
			}

			res := c.Response()
			res.Header().Set(HeaderXTraceID, traceID)

			start := time.Now()
			if err := next(c); err != nil {
				c.Error(err)
			}
			end := time.Now()

			path := c.Path()
			if path == "" {
				path = req.URL.Path
			}

			attrs := []any{
				slog.String("trace_id", traceID),
				slog.String("protocol", req.Proto),
				slog.String("method", req.Method),
				slog.String("path", path),
				slog.String("host", req.Host),
				slog.String("client_ip", c.RealIP()),
				slog.Int("status", res.Status),
				slog.Int64("bytes_out", res.Size),
				slog.Int64("duration_ms", end.Sub(start).Milliseconds()),
			}

			if qp := c.QueryParams(); len(qp) > 0 {
				attrs = append(attrs, slog.Any("query_params", qp))
			}
			if cr != nil {
				attrs = append(attrs, slog.Int64("bytes_in", cr.n))
			}

			pathParamNames := c.ParamNames()
			if len(pathParamNames) > 0 {
				pathParams := make(map[string]string, len(pathParamNames))
				for _, name := range pathParamNames {
					pathParams[name] = c.Param(name)
				}
				attrs = append(attrs, slog.Any("path_params", pathParams))
			}

			clientID := req.Header.Get(HeaderXClientID)
			if clientID != "" {
				attrs = append(attrs, slog.String("client_id", clientID))
			}

			if user, ok := c.Get("user").(*repository.User); ok && (user != nil) {
				attrs = append(attrs, slog.String("user_id", user.ID.String()))
			}

			logger.Info("HTTP Request", attrs...)

			return nil
		}
	}
}
