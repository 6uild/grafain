package webhook

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/iov-one/weave/app"
	"github.com/iov-one/weave/errors"
	"github.com/tendermint/tendermint/libs/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// Start webhook server and block.
func Start(mgr manager.Manager, serverAddress string, certDir, admissionPath string, store <-chan *app.StoreApp, logger log.Logger) error {
	logger.Info("Setting up webhook server")
	hookServer := mgr.GetWebhookServer()
	hookServer.CertDir = certDir
	parts := strings.Split(serverAddress, ":")
	if len(parts) != 2 {
		return errors.Wrapf(errors.ErrInput, "Invalid address :%q", serverAddress)
	}
	var err error
	hookServer.Host = parts[0]
	hookServer.Port, err = strconv.Atoi(parts[1])
	if err != nil {
		return errors.Wrapf(errors.ErrInput, "Invalid port :%q", parts[1])
	}

	logger.Info("Registering webhooks to the internal webhook server")
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

	logger.Info("Starting manager", "address", serverAddress)
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		return errors.Wrap(err, "Unable to run manager")
	}
	logger.Info("Stopped")
	return nil
}
