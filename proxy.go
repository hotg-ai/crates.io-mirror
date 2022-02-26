package main

import (
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"time"

	"go.uber.org/zap"
)

func Handler(logger *zap.Logger, upstream *url.URL) http.Handler {
	mux := http.NewServeMux()

	proxied := proxy(upstream)

	mux.HandleFunc("/", proxied)

	return logged(logger, mux)
}

// proxy a request to another server.
//
// Mostly copied from https://gist.github.com/yowu/f7dc34bd4736a65ff28d
func proxy(upstream *url.URL) http.HandlerFunc {
	jar, _ := cookiejar.New(nil)

	client := http.Client{
		Timeout: 60 * time.Second,
		Jar:     jar,
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

		logger.Debug(
			"Receiving response from upstream",
			zap.Any("headers", response.Header),
		)

		w.WriteHeader(response.StatusCode)
		if _, err := io.Copy(w, response.Body); err != nil {
			logger.Error("Unable to write the response", zap.Error(err))
		}
	}
}
