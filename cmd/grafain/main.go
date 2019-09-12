package main

import (
	"context"
	"crypto/tls"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alpe/grafain/pkg/xadmission"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
)

// application version, will be set during compilation time
var version string

func main() {
	var (
		serverAddress = flag.String("server-port", "0.0.0.0:8443", "Server address for incoming connections.")
		tlsCertFile   = flag.String("tls-cert", "/certs/tls.crt", "TLS certificate file.")
		tlsKeyFile    = flag.String("tls-key", "/certs/tls.key", "TLS key file.")
	)
	flag.Parse()
	logger := log.NewJSONLogger(log.NewSyncWriter(os.Stdout))
	logger = log.With(logger, "ts", log.DefaultTimestampUTC, "caller", log.DefaultCaller, "version", version)
	if len(*serverAddress) == 0 {
		level.Error(logger).Log("message", "Server address must not be empty")
		os.Exit(1)
	}
	if len(*tlsCertFile) == 0 {
		level.Error(logger).Log("message", "tls-cert must not be empty")
		os.Exit(1)
	}
	if len(*tlsKeyFile) == 0 {
		level.Error(logger).Log("message", "tls-key must not be empty")
		os.Exit(1)
	}

	defer recoverToLog(logger)
	mux := http.NewServeMux()
	mux.Handle("/healthz", NoOpHandler())
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		xadmission.ReviewHandler(w, r, log.With(logger, "component", "hook"))
	})

	svr := &http.Server{
		Addr:    *serverAddress,
		Handler: mux,
		TLSConfig: &tls.Config{
			ClientAuth: tls.NoClientCert,
		},
	}
	go func() {
		level.Info(logger).Log("message", "server started", "address", svr.Addr)
		if err := svr.ListenAndServeTLS(*tlsCertFile, *tlsKeyFile); err != nil {
			level.Error(logger).Log("message", "server error", "cause", err, "address", svr.Addr)
			os.Exit(10)
		}
	}()
	awaitGracefulShutdown(svr, logger, 9*time.Second)
}

func recoverToLog(logger log.Logger) {
	if err := recover(); err != nil {
		level.Error(logger).Log("message", "Recover from panic", "cause", err)
		os.Exit(1)
	}
}

func awaitGracefulShutdown(svr *http.Server, logger log.Logger, timeout time.Duration) {
	gracefulStop := make(chan os.Signal, 1)
	signal.Notify(gracefulStop, os.Interrupt, syscall.SIGTERM)
	<-gracefulStop

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	logger.Log("info", "Server shutdown", "timeout", timeout)

	if err := svr.Shutdown(ctx); err != nil {
		logger.Log("error", "server error", "cause", err)
	} else {
		logger.Log("info", "Server stopped")
	}
}

// NoOpHandler returns 200 code only. This handler can be used for handling probe requests.
func NoOpHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}
