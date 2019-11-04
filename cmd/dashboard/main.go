package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"flag"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/alpe/grafain/pkg/artifact"
	"github.com/alpe/grafain/pkg/client"
	"github.com/iov-one/weave"
	weaveclient "github.com/iov-one/weave/client"
	"github.com/tendermint/tendermint/libs/log"
)

func main() {
	addr := flag.String("listen-address", ":8081", "Server address with host and port: default:0.0.0.0:8081")
	tmAddress := flag.String("tm-address", env("GRAFAINCLI_TM_ADDR", "https://grafain.NETWORK.iov.one:443"), "Tendermint node address. Use proper NETWORK name. You can use GRAFAINCLI_TM_ADDR environment variable to set it.")
	flag.Parse()

	logger := log.NewTMLogger(log.NewSyncWriter(os.Stdout))

	if len(*addr) == 0 {
		logger.Error("address must not be empty")
		os.Exit(2)
	}
	srv := setupHttpServer(addr, tmAddress, logger)

	done := SetupSignalHandler()
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			logger.Error("server stopped", "cause", err)
			os.Exit(2)
		}
	}()
	<-done
	if err := srv.Shutdown(context.Background()); err != nil {
		logger.Error("HTTP server shutdown", "cause", err)
		os.Exit(2)

	}
	logger.Info("Done")
}

func setupHttpServer(addr *string, tmAddress *string, logger log.Logger) *http.Server {
	router := http.NewServeMux()
	router.Handle("/css/", http.StripPrefix("/css/", http.FileServer(http.Dir(filePath("/css")))))
	router.Handle("/images/", http.StripPrefix("/images/", http.FileServer(http.Dir(filePath("/images")))))
	router.HandleFunc("/", board(client.NewClient(weaveclient.NewHTTPConnection(*tmAddress)), logger))
	srv := &http.Server{
		Addr:         *addr,
		Handler:      router,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}
	return srv
}

type keyval struct {
	Key   string
	Value artifact.Artifact
}

func board(grafainClient *client.Client, logger log.Logger) func(http.ResponseWriter, *http.Request) {
	boardTemplate := template.Must(template.ParseFiles(filePath("index.html")))
	return func(resp http.ResponseWriter, req *http.Request) {
		artfs, err := query(grafainClient)
		if err != nil {
			logger.Info("Failed to query artifacts", "cause", err)
			resp.WriteHeader(http.StatusInternalServerError)
			return
		}
		logger.Debug("Query result", "count", len(artfs))
		data := struct {
			Items []keyval
		}{
			artfs,
		}

		if err := boardTemplate.Execute(resp, data); err != nil {
			logger.Info("Failed to render page", "cause", err)
			resp.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
}

func filePath(s string) string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), s)
}

func query(grafainClient *client.Client) ([]keyval, error) {
	resp, err := grafainClient.AbciQuery("/artifacts?"+weave.PrefixQueryMod, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to run query: %s", err)
	}
	result := make([]keyval, len(resp.Models))
	for i, m := range resp.Models {
		var obj artifact.Artifact
		if err := obj.Unmarshal(m.Value); err != nil {
			return nil, fmt.Errorf("failed to unmarshal model %d: %s", i, err)
		}
		key, err := sequenceKey(m.Key)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal key %d: %s", i, err)
		}
		result[i] = keyval{Key: key, Value: obj}
	}
	return result, nil
}

func sequenceKey(raw []byte) (string, error) {
	// Skip the prefix, being the characters before : (including separator)
	seq := raw[bytes.Index(raw, []byte(":"))+1:]
	if len(seq) != 8 {
		return "", fmt.Errorf("invalid sequence length: %d", len(seq))
	}
	n := binary.BigEndian.Uint64(seq)
	return fmt.Sprint(int64(n)), nil
}

func SetupSignalHandler() (stopCh <-chan struct{}) {
	stop := make(chan struct{})
	done := make(chan os.Signal, 2)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-done
		close(stop)
		<-done
		os.Exit(1) // second signal. Exit directly.
	}()

	return stop
}

// env returns the value of an environment variable if provided (even if empty)
// or a fallback value.
func env(name, fallback string) string {
	if v, ok := os.LookupEnv(name); ok {
		return v
	}
	return fallback
}
