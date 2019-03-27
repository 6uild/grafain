package main

import (
	"flag"
	"github.com/alpe/grafain/pkg/server"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"net/http"
	"os"
	"time"
)

// application version, will be set during compilation time
var version string

func main() {
	var (
		serverAddress = flag.String("server-port", "0.0.0.0:8080", "server address for incoming connections")
	)
	flag.Parse()
	logger := log.NewJSONLogger(log.NewSyncWriter(os.Stdout))
	logger = log.With(logger, "ts", log.DefaultTimestampUTC, "caller", log.DefaultCaller, "version", version)
	if len(*serverAddress) == 0 {
		level.Error(logger).Log("message", "Server address must not be empty")
		os.Exit(1)
	}

	defer recoverToLog(logger)
	mux := http.NewServeMux()

	mux.Handle("/ping", server.NoOpHandler())
	mux.Handle("/hook", server.NewWebhookServer(log.With(logger, "component", "hook")))

	svr := &http.Server{Addr: *serverAddress, Handler: mux}
	go server.Run(svr, logger)
	server.AwaitGracefulShutdown(svr, logger, 9*time.Second)
}

func recoverToLog(logger log.Logger) {
	if err := recover(); err != nil {
		level.Error(logger).Log("message", "Recover from panic", "cause", err)
		os.Exit(1)
	}
}

