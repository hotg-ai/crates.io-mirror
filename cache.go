package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"go.uber.org/zap"
)

type Cache interface {
	Get(logger *zap.Logger, path string) ([]byte, bool)
	Update(logger *zap.Logger, path string, content []byte) error
}

func cached(c Cache, handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := getLogger(r)
		path := r.URL.EscapedPath()
		content, ok := c.Get(logger, path)

		if ok {
			logger.Info(
				"Serving up a cached response",
				zap.Any("bytes", len(content)),
				zap.Any("path", path),
			)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(content)
			return
		}

		buffer := bytes.Buffer{}

		// Call the original handler and save the response to a buffer
		tee := teeResponseWriter{
			inner:  w,
			code:   http.StatusOK,
			writer: io.MultiWriter(&buffer, w),
		}
		handler(&tee, r)

		if tee.code != http.StatusOK {
			logger.Info(
				"Not caching the result because the server didn't reply with a 200 OK",
				zap.Any("status-code", tee.code),
				zap.Any("status-text", http.StatusText(tee.code)),
			)
			return
		}

		if err := c.Update(logger, path, buffer.Bytes()); err != nil {
			logger.Warn(
				"Unable to update the cache",
				zap.Error(err),
				zap.Any("path", path),
			)
		}
	}
}

type teeResponseWriter struct {
	inner  http.ResponseWriter
	code   int
	writer io.Writer
}

func (t *teeResponseWriter) Header() http.Header {
	return t.inner.Header()
}

func (t *teeResponseWriter) WriteHeader(code int) {
	t.code = code
	t.inner.WriteHeader(code)
}

func (t *teeResponseWriter) Write(data []byte) (int, error) {
	return t.writer.Write(data)
}

func newS3Cache(bucket string) (Cache, error) {
	session, err := session.NewSession()
	if err != nil {
		return nil, err
	}

	uploader := s3manager.NewUploader(session)
	downloader := s3manager.NewDownloader(session)

	return &s3Cache{bucket, uploader, downloader}, nil
}

type s3Cache struct {
	bucket string
	up     *s3manager.Uploader
	down   *s3manager.Downloader
}

func (s *s3Cache) Get(logger *zap.Logger, path string) ([]byte, bool) {
	w := aws.NewWriteAtBuffer(nil)
	_, err := s.down.Download(w, &s3.GetObjectInput{Bucket: &s.bucket})

	if err != nil {
		return nil, false
	}

	return w.Bytes(), true
}

func (s *s3Cache) Update(logger *zap.Logger, path string, content []byte) error {
	panic("TODO")
}

func newLocalCache(dir string) (*localCache, error) {
	dir, err := filepath.Abs(dir)

	if err != nil {
		return nil, err
	}

	if err = os.MkdirAll(dir, 0o744); err != nil {
		return nil, fmt.Errorf("unable to create the cache directory: %w", err)
	}

	return &localCache{dir}, nil
}

type localCache struct {
	baseDir string
}

var ErrOutsideBaseDirectory = errors.New("the resulting path is outside the local cache's base directory")

func (l *localCache) fullPath(p string) (string, error) {
	joined := path.Join(l.baseDir, p)

	if !strings.HasPrefix(joined, l.baseDir) {
		return "", ErrOutsideBaseDirectory
	}

	return joined, nil
}

func (l *localCache) Get(logger *zap.Logger, path string) (content []byte, ok bool) {
	path, err := l.fullPath(path)

	if err != nil {
		logger.Warn(
			"Unable to determine the full path", zap.Error(err),
			zap.String("path", path),
			zap.String("base-dir", l.baseDir),
		)
		return nil, false
	}

	content, err = ioutil.ReadFile(path)

	if os.IsNotExist(err) {
		return nil, false
	} else if err != nil {
		logger.Warn(
			"Unable to read the cached file",
			zap.Error(err),
			zap.String("path", path),
		)
		return nil, false
	}

	return content, true
}

func (l *localCache) Update(logger *zap.Logger, p string, content []byte) error {
	p, err := l.fullPath(p)
	if err != nil {
		return err
	}

	parent := path.Dir(p)

	if err = os.MkdirAll(parent, 0o744); err != nil {
		return fmt.Errorf("unable to create the %s directory: %w", parent, err)
	}

	err = ioutil.WriteFile(p, content, 0o644)

	if err == nil {
		logger.Debug(
			"Cache updated",
			zap.Any("path", p),
			zap.Any("bytes", len(content)),
		)
	}

	return err
}
