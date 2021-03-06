package main

import (
	"io"
	"net/http"
	"net/url"
	"time"

	"go.uber.org/zap"
)

// proxy returns a http.HandlerFunc that passes all requests through to another
// server.
func proxy(upstream *url.URL) http.HandlerFunc {
	client := http.Client{
		Timeout: 60 * time.Second,
	}

	return func(w http.ResponseWriter, r *http.Request) {
		logger := getLogger(r)

		r.URL.Scheme = upstream.Scheme
		r.URL.Host = upstream.Host

		req, err := http.NewRequestWithContext(r.Context(), r.Method, r.URL.String(), r.Body)
		if err != nil {
			http.Error(w, "Server Error", http.StatusInternalServerError)
			logger.Error(
				"Unable to create the new request",
				zap.Error(err),
			)
			return
		}
		req.Header = r.Header

		// Telling the client to upgrade to HTTP2 seems to fail, so let's remove
		// these headers for now
		req.Header.Del("Upgrade")
		req.Header.Del("Connection")

		logger.Debug(
			"Proxying request to upstream",
			zap.Any("request-headers", req.Header),
			zap.Any("url", req.URL),
			zap.Any("host", req.Host),
			zap.Any("referer", req.Referer()),
		)

		response, err := client.Do(req)
		if err != nil {
			http.Error(w, "Server Error", http.StatusInternalServerError)
			logger.Error(
				"Unable to send the request",
				zap.Error(err),
			)
			return
		}
		defer response.Body.Close()

		w.WriteHeader(response.StatusCode)
		if _, err := io.Copy(w, response.Body); err != nil {
			logger.Error("Unable to write the response", zap.Error(err))
		}
	}
}
