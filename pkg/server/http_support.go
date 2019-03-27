package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/go-kit/kit/log"
)

func Run(server *http.Server, logger log.Logger) {
	logger.Log("event", "server started", "address", server.Addr)
	if err := server.ListenAndServe(); err != nil {
		logger.Log("error", "server error", "cause", err, "address", server.Addr)
		os.Exit(-2)
	}
}

func AwaitGracefulShutdown(svr *http.Server, logger log.Logger, timeout time.Duration) {
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


func RespondJson(w http.ResponseWriter, code int, content interface{}) error {
	var buf bytes.Buffer
	err := json.NewEncoder(&buf).Encode(content)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", JSONContentType)
	w.Header().Set("Content-Length", strconv.Itoa(buf.Len()))
	w.WriteHeader(code)
	_, _ = io.Copy(w, &buf)
	return nil
}
