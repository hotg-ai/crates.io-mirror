package main

import (
	"context"
	"fmt"
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

		logger := rootLogger.With(zap.Any("request-id", requestId))
		r = r.WithContext(context.WithValue(r.Context(), loggerKey{}, logger))

		defer func() {
			if err := recover(); err != nil {
				logger.Error(
					"Handler panicked",
					zap.Any("error", err),
					zap.Stack("stack"),
				)
			}
		}()

		spy := writer{inner: w, code: http.StatusOK}
		start := time.Now()

		handler.ServeHTTP(&spy, r)

		duration := time.Since(start)

		logger.Info(
			"Served a request",
			zap.Any("response-code", spy.code),
			zap.Any("bytes-written", spy.bytesWritten),
			zap.Any("url", r.URL),
			zap.Any("method", r.Method),
			zap.Any("headers", r.Header),
			zap.Any("duration", duration),
			zap.Any("user-agent", r.UserAgent()),
		)
	})
}

type loggerKey struct{}

// getLogger returns the logger specific to this request.
func getLogger(r *http.Request) *zap.Logger {
	logger, ok := r.Context().Value(loggerKey{}).(*zap.Logger)

	if !ok {
		msg := fmt.Sprintf("Attempted to get the request logger when the handler doesn't have one: %s", r.URL)
		panic(msg)
	}

	return logger
}

type writer struct {
	inner        http.ResponseWriter
	bytesWritten int
	code         int
}

func (w *writer) Header() http.Header {
	return w.inner.Header()
}

func (w *writer) Write(data []byte) (int, error) {
	bytesWritten, err := w.inner.Write(data)
	w.bytesWritten += bytesWritten
	return bytesWritten, err
}

func (w *writer) WriteHeader(statusCode int) {
	w.code = statusCode
	w.inner.WriteHeader(statusCode)
}
