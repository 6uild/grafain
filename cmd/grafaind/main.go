package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	grafain "github.com/alpe/grafain/pkg/app"
	"github.com/alpe/grafain/pkg/webhook"
	"github.com/iov-one/weave/commands/server"
	"github.com/iov-one/weave/errors"
	"github.com/tendermint/tendermint/libs/log"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
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
		tmRPCAddress  = flag.String("rpc-address", "http://127.0.0.1:26657", "Tendermint node address.")
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
		awaitErr := make(chan error, 2)
		go func() {
			awaitErr <- startWebHook(*tmRPCAddress, *hookAddress, *certDir, *admissionPath, logger.With("module", "admission-hook"))
		}()
		go func() {
			awaitErr <- server.StartCmd(grafain.GenerateApp, logger, *varHome, rest)
		}()
		err = <-awaitErr
	case "getblock":
		err = server.GetBlockCmd(rest)
	case "retry":
		err = server.RetryCmd(grafain.InlineApp, logger, *varHome, rest)
	case "version":
		fmt.Println(version)
	default:
		err = fmt.Errorf("unknown command: %s", cmd)
		helpMessage()
	}

	if err != nil {
		fmt.Printf("Error: %+v\n\n", err)
		os.Exit(1)
	}
}

func startWebHook(tmRPCAddress, hookServerAddress string, certDir, admissionPath string, logger log.Logger) error {
	logger.Debug("Setting up manager")
	cfg, err := config.GetConfig()
	if err != nil {
		return errors.Wrap(err, "Unable to load kubernetes config")
	}
	mgr, err := manager.New(cfg, manager.Options{})
	if err != nil {
		return errors.Wrap(err, "Unable to set up overall controller manager")
	}
	return webhook.Start(mgr, tmRPCAddress, hookServerAddress, certDir, admissionPath, logger)
}
