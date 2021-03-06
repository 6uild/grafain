package webhook

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/alpe/grafain/pkg/client"
	weaveclient "github.com/iov-one/weave/client"
	"github.com/iov-one/weave/errors"
	"github.com/tendermint/tendermint/libs/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// Start webhook server and block.
func Start(mgr manager.Manager, rpcAddress, hookServerAddress string, certDir, admissionPath string, logger log.Logger) error {
	logger.Info("Setting up webhook server")
	hookServer := mgr.GetWebhookServer()
	hookServer.CertDir = certDir
	parts := strings.Split(hookServerAddress, ":")
	if len(parts) != 2 {
		return errors.Wrapf(errors.ErrInput, "Invalid address :%q", hookServerAddress)
	}
	var err error
	hookServer.Host = parts[0]
	hookServer.Port, err = strconv.Atoi(parts[1])
	if err != nil {
		return errors.Wrapf(errors.ErrInput, "Invalid port :%q", parts[1])
	}

	grafainClient := client.NewClient(weaveclient.NewHTTPConnection(rpcAddress))

	logger.Info("Registering webhooks to the internal webhook server")
	hookServer.Register(admissionPath, &webhook.Admission{
		Handler: NewPodValidator(
			grafainClient,
			logger.With("module", "pod-validator"),
		),
	})
	hookServer.Register("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("Admission hook: " + admissionPath))
	}))

	logger.Info("Starting manager", "address", hookServerAddress)
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		return errors.Wrap(err, "Unable to run manager")
	}
	logger.Info("Stopped")
	return nil
}
