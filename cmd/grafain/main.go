package main

import (
	"os"

	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// Proof of concept without controller handling
// - reconciler
// - watch
func main() {
	var logger = logf.Log.WithName("grafain")
	logf.SetLogger(zap.Logger(true))

	logger.Info("setting up manager")
	mgr, err := manager.New(config.GetConfigOrDie(), manager.Options{})

	if err != nil {
		logger.Error(err, "unable to set up overall controller manager")
		os.Exit(1)
	}

	logger.Info("setting up webhook server")
	hookServer := mgr.GetWebhookServer()
	hookServer.CertDir = "/certs"
	hookServer.Port = 8443

	logger.Info("registering webhooks to the webhook server")
	hookServer.Register("/validate-v1-pod", &webhook.Admission{
		Handler: &podValidator{logger: logger.WithName("hook")},
	})

	logger.Info("starting manager")
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		logger.Error(err, "unable to run manager")
		os.Exit(1)
	}
	logger.Info("done")
}
