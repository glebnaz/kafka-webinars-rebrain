package main

import (
	"context"
	"google.golang.org/grpc"
	"os"

	grpcInternal "github.com/glebnaz/go-platform/grpc"
	log "github.com/sirupsen/logrus"
)

func init() {
	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(os.Stdout)
	level, err := log.ParseLevel(os.Getenv("LOGLVL"))
	if err != nil {
		level = log.DebugLevel
	}
	log.SetLevel(level)
}

func main() {
	ctx := context.Background()

	app := newApp(ctx, grpc.UnaryInterceptor(grpcInternal.NewServerUnaryLoggerInterceptor()))

	if err := app.Start(ctx); err != nil {
		log.Errorf("bad start app: %s", err)
	}
}
