package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
)

var ErrOutsideBaseDirectory = errors.New("the resulting path is outside the local cache's base directory")

type localCache struct {
	baseDir string
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
