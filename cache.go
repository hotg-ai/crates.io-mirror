package main

import (
	"io"
	"net/http"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"go.uber.org/zap"
)

type Cache interface {
}

func cached(c Cache, handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		panic("TODO")
	}
}

func newS3Cache(logger *zap.Logger, bucket string) Cache {
	session, err := session.NewSession()
	if err != nil {
		logger.Fatal("Unable to create the AWS session", zap.Error(err))
	}

	uploader := s3manager.NewUploader(session)
	downloader := s3manager.NewDownloader(session)

	return &s3Cache{bucket, uploader, downloader}
}

type s3Cache struct {
	bucket string
	up     *s3manager.Uploader
	down   *s3manager.Downloader
}

func (s *s3Cache) Get(path string) ([]byte, bool) {
	w := aws.NewWriteAtBuffer(nil)
	_, err := s.down.Download(w, &s3.GetObjectInput{Bucket: &s.bucket})

	if err != nil {
		return nil, false
	}

	return w.Bytes(), true
}

func (s *s3Cache) Put(path string) io.WriteCloser {
	panic("TODO")
}
