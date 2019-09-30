package grafain

import (
	"context"
	"path/filepath"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/app"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/commands/server"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/x/cash"
	"github.com/iov-one/weave/x/currency"
	"github.com/iov-one/weave/x/distribution"
	"github.com/iov-one/weave/x/escrow"
	"github.com/iov-one/weave/x/gov"
	"github.com/iov-one/weave/x/msgfee"
	"github.com/iov-one/weave/x/multisig"
	"github.com/iov-one/weave/x/validators"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/log"
)

// GenerateApp is used to create a stub for server/start.go command
func GenerateApp(options *server.Options) (abci.Application, error) {
	// db goes in a subdir, but "" stays "" to use memdb
	var dbPath string
	if options.Home != "" {
		dbPath = filepath.Join(options.Home, "grafain.db")
	}

	stack := Stack(nil, options.MinFee)
	application, err := Application("grafain", stack, TxDecoder, dbPath, options)
	if err != nil {
		return nil, err
	}
	return DecorateApp(application, options.Logger), nil
}

// DecorateApp adds initializers and Logger to an Application
func DecorateApp(application app.BaseApp, logger log.Logger) app.BaseApp {
	application.WithInit(app.ChainInitializers(
		&migration.Initializer{},
		&multisig.Initializer{},
		&cash.Initializer{},
		&currency.Initializer{},
		&validators.Initializer{},
		&distribution.Initializer{},
		&msgfee.Initializer{},
		&escrow.Initializer{Minter: cash.NewController(cash.NewBucket())},
		&gov.Initializer{},
	))
	application.WithLogger(logger)
	return application
}

// InlineApp will take a previously prepared CommitStore and return a complete Application
func InlineApp(kv weave.CommitKVStore, logger log.Logger, debug bool) abci.Application {
	minFee := coin.Coin{}
	stack := Stack(nil, minFee)
	ctx := context.Background()
	store := app.NewStoreApp("bnsd", kv, QueryRouter(minFee), ctx)
	base := app.NewBaseApp(store, TxDecoder, stack, nil, debug)
	return DecorateApp(base, logger)
}
