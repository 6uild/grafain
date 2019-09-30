package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	grafain "github.com/alpe/grafain/cmd/grafain/app"
	"github.com/alpe/grafain/cmd/grafain/xwebhook"
	"github.com/iov-one/weave/app"
	"github.com/iov-one/weave/commands/server"
	"github.com/iov-one/weave/errors"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/log"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
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
		appGenFactory, storage := appWithStorage()
		go xwebhook.Start(*hookAddress, *certDir, *admissionPath, storage, logger.With("module", "admission-hook"))
		err = server.StartCmd(appGenFactory, logger, *varHome, rest)
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

// the abci server is started with an application factory method.
// within this method the storage is initialized. This function wraps
// the factory method and sends the storage object to the returned channel to
// allow its usage ouside of the abci server context.
func appWithStorage() (func(options *server.Options) (abci.Application, error), chan *app.StoreApp) {
	c := make(chan *app.StoreApp)
	appGenHack := func(options *server.Options) (abci.Application, error) {
		gApp, err := grafain.GenerateApp(options)
		if err != nil {
			return gApp, errors.Wrap(err, "failed to init grafain app")
		}
		c <- gApp.(app.BaseApp).StoreApp
		close(c)
		return gApp, err
	}
	return appGenHack, c
}
