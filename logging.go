package main

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// logged is a middleware that logs some basic information about a response
// (url, status code, response time, etc.) and attaches a request-specific
// logger to the request's context.
//
// Panics will be automatically logged.
func logged(rootLogger *zap.Logger, handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestId := uuid.New()

		// Create our specialized logger and attach it to the request
		logger := rootLogger.With(zap.Any("request-id", requestId))
		r = r.WithContext(context.WithValue(r.Context(), loggerKey{}, logger))

		defer func() {
			if err := recover(); err != nil {
				logger.Error(
					"Handler panicked",
					zap.Any("error", err),
					zap.StackSkip("stack", 1),
				)
			}
		}()

		spy := spyWriter{inner: w, code: http.StatusOK}
		start := time.Now()

		handler.ServeHTTP(&spy, r)

		duration := time.Since(start)

		logger.Info(
			"Served a request",
			zap.Any("status-code", spy.code),
			zap.Any("status-text", http.StatusText(spy.code)),
			zap.Any("bytes-written", spy.bytesWritten),
			zap.Any("url", r.URL),
			zap.Any("method", r.Method),
			zap.Any("duration", duration),
			zap.Any("user-agent", r.UserAgent()),
			zap.Any("remote-addr", r.RemoteAddr),
		)
	})
}

type loggerKey struct{}

// getLogger returns the logger specific to this request.
func getLogger(r *http.Request) *zap.Logger {
	logger, ok := r.Context().Value(loggerKey{}).(*zap.Logger)

	if !ok {
		// fall back to the global logger
		return zap.L()
	}

	return logger
}

type spyWriter struct {
	inner        http.ResponseWriter
	bytesWritten int
	code         int
}

func (w *spyWriter) Header() http.Header {
	return w.inner.Header()
}

func (w *spyWriter) Write(data []byte) (int, error) {
	bytesWritten, err := w.inner.Write(data)
	w.bytesWritten += bytesWritten
	return bytesWritten, err
}

func (w *spyWriter) WriteHeader(statusCode int) {
	w.code = statusCode
	w.inner.WriteHeader(statusCode)
}
