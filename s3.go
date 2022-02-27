package main

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"go.uber.org/zap"
)

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
