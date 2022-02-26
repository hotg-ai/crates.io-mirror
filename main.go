package main

import (
	"log"
	"os"

	"github.com/jessevdk/go-flags"
	"go.uber.org/zap"
)

func main() {
	opts := opts{}

	_, err := flags.Parse(&opts)
	if err != nil {
		os.Exit(1)
	}

	logger := opts.logger()

	logger.Info("Started", zap.Any("args", opts))
}

type opts struct {
	Verbose bool `short:"v" long:"verbose" description:"Show more verbose debug information"`
}

func (o opts) logger() *zap.Logger {
	var config zap.Config

	if o.Verbose {
		config = zap.NewDevelopmentConfig()
	} else {
		config = zap.NewProductionConfig()
	}

	logger, err := config.Build()
	if err != nil {
		log.Fatalf("Unable to initialize the logger: %v", err)
	}

	return logger
}
