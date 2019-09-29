package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	grafain "github.com/alpe/grafain/cmd/grafain/app"
	"github.com/iov-one/weave/commands/server"
	"github.com/tendermint/tendermint/libs/log"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

var (
	version string
)

func helpMessage() {
	fmt.Println("grafain")
	fmt.Println("          Custom Blockchain Service node")
	fmt.Println("")
	fmt.Println("help      Print this message")
	fmt.Println("start     Run the abci server")
	fmt.Println("getblock  Extract a block from blockchain.db")
	fmt.Println("retry     Run last block again to ensure it produces same result")
	fmt.Println("version   Print the app version")
	fmt.Println(`
  -home string
        directory to store files under (default "$HOME/.grafain")`)
}

func main() {
	defaultHome := filepath.Join(os.ExpandEnv("$HOME"), ".grafain")
	var (
		varHome       = flag.String("home", defaultHome, "directory to store files under")
		certDir       = flag.String("hook-certs", "/certs", "TLS certrificates")
		hookAddress   = flag.String("hook-address", ":8443", "Webhook server address with host and port. default: 0.0.0.0:8443")
		admissionPath = flag.String("hook-path", "/validate-v1-pod", "Url path for admission hook. default: /validate-v1-pod")
	)
	flag.CommandLine.Usage = helpMessage

	flag.Parse()
	if flag.NArg() == 0 {
		fmt.Println("Missing command:")
		helpMessage()
		os.Exit(1)
	}

	cmd := flag.Arg(0)
	rest := flag.Args()[1:]
	fmt.Println(rest)

	logger := log.NewTMLogger(log.NewSyncWriter(os.Stdout))

	var err error
	switch cmd {
	case "help":
		helpMessage()
	case "start":
		go startHook(*hookAddress, *certDir, *admissionPath, logger.With("module", "admission-hook"))
		err = server.StartCmd(grafain.GenerateApp, logger, *varHome, rest)
	case "getblock":
		err = server.GetBlockCmd(rest)
	case "retry":
		err = server.RetryCmd(grafain.InlineApp, logger, *varHome, rest)
	case "version":
		fmt.Println(version)
	default:
		err = fmt.Errorf("unknown command: %s", cmd)
	}

	if err != nil {
		fmt.Printf("Error: %+v\n\n", err)
		helpMessage()
		os.Exit(1)
	}
}

const admissionPath = ""

func startHook(serverAddress string, certDir, admissionPath string, logger log.Logger) {
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
		Handler: &podValidator{logger: logger.With("module", "pod-validator")},
	})
	hookServer.Register("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Admission hook: " + admissionPath))
	}))

	logger.Info("starting manager", "address", serverAddress)
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		logger.Error("unable to run manager", "cause", err)
		os.Exit(1)
	}
	logger.Info("stopped")
}
