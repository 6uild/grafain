package xwebhook

import (
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/iov-one/weave/app"
	"github.com/tendermint/tendermint/libs/log"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

func Start(serverAddress string, certDir, admissionPath string, store <-chan *app.StoreApp, logger log.Logger) {
	logger.Info("setting up manager")
	mgr, err := manager.New(config.GetConfigOrDie(), manager.Options{})

	if err != nil {
		logger.Error("unable to set up overall controller manager", "cause", err)
		os.Exit(1)
	}

	logger.Info("setting up webhook server")
	hookServer := mgr.GetWebhookServer()
	hookServer.CertDir = certDir
	parts := strings.Split(serverAddress, ":")
	if len(parts) != 2 {
		logger.Error("invalid address", "address", serverAddress)
		os.Exit(1)
	}
	hookServer.Host = parts[0]
	hookServer.Port, err = strconv.Atoi(parts[1])
	if err != nil {
		logger.Error("invalid port", "port", parts[1], "cause", err)
		os.Exit(1)
	}

	logger.Info("registering webhooks to the webhook server")
	hookServer.Register(admissionPath, &webhook.Admission{
		Handler: NewPodValidator(
			<-store,
			logger.With("module", "pod-validator"),
		),
	})
	hookServer.Register("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Admission hook: " + admissionPath))
	}))

	logger.Info("starting manager", "address", serverAddress)
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		logger.Error("unable to run manager", "cause", err)
		os.Exit(1)
	}
	logger.Info("stopped")
}
