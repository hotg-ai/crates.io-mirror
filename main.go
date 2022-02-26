package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/jessevdk/go-flags"
	"go.uber.org/zap"
)

func main() {
	opts := opts{
		Host:     "localhost",
		Port:     8080,
		Upstream: "https://crates.io/",
	}

	_, err := flags.Parse(&opts)
	if err != nil {
		os.Exit(1)
	}

	logger := opts.logger()

	logger.Info("Started", zap.Any("args", opts))

	upstream, err := url.Parse(opts.Upstream)
	if err != nil {
		logger.Fatal("Unable to parse the upstream URL", zap.Error(err), zap.String("upstream", opts.Upstream))
	}

	addr := fmt.Sprintf("%s:%d", opts.Host, opts.Port)

	server := http.Server{
		Addr:    addr,
		Handler: Handler(logger, upstream),
	}

	logger.Info("Serving", zap.Any("addr", addr))
	go shutdownOnCtrlC(logger, &server)

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Fatal("Unable to start the server", zap.Error(err))
	}
}

type opts struct {
	Verbose  bool   `short:"v" long:"verbose" description:"Show more verbose debug information" env:"VERBOSE"`
	Upstream string `short:"u" long:"upstream" description:"The URL to proxy requests to" env:"UPSTREAM"`
	Host     string `short:"H" long:"host" description:"The interface to listen on" env:"HOST"`
	Port     int    `short:"p" long:"port" description:"The port to use" env:"PORT"`
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

	zap.RedirectStdLog(logger)

	return logger
}

func shutdownOnCtrlC(logger *zap.Logger, s *http.Server) {
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt)

	<-done

	logger.Info("Shutting down")

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	if err := s.Shutdown(ctx); err != nil {
		logger.Fatal("Unable to shutdown", zap.Error(err))
	}
}
