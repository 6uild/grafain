package main

// In this file resides implementation a lot of private functionality from the
// tendermint/rpc/test package. We cannot use this package directly, because it
// relies on a global configuration state. We must provide a unique
// configuration for each test case, therefore we must provide our own test
// helpers.
//
// This file contains mostly copy/paste code with small adjustments to not rely
// on the global configuration.

import (
	"context"
	"fmt"
	"testing"
	"time"

	abci "github.com/tendermint/tendermint/abci/types"
	cfg "github.com/tendermint/tendermint/config"
	cmn "github.com/tendermint/tendermint/libs/common"
	"github.com/tendermint/tendermint/libs/log"
	nm "github.com/tendermint/tendermint/node"
	"github.com/tendermint/tendermint/p2p"
	"github.com/tendermint/tendermint/privval"
	"github.com/tendermint/tendermint/proxy"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
	core_grpc "github.com/tendermint/tendermint/rpc/grpc"
	rpcclient "github.com/tendermint/tendermint/rpc/lib/client"
)

func buildTendermintConfig(t testing.TB, workDir string) *cfg.Config {
	// Do not use rpctest.GetConfig as it is using a global variable to
	// cache the configuration. This configuration must be unique per test
	// as our application must always get an empty database directory
	// during the initialization.
	c := cfg.ResetTestRoot(workDir)
	c.P2P.ListenAddress = randomAddr(t)
	c.RPC.ListenAddress = randomAddr(t)
	c.RPC.CORSAllowedOrigins = []string{"https://tendermint.com/"}
	c.RPC.GRPCListenAddress = randomAddr(t)
	c.TxIndex.IndexTags = "app.creator,tx.height"
	return c
}

func waitForRPC(t testing.TB, c *cfg.Config) {
	t.Helper()
	laddr := c.RPC.ListenAddress
	client := rpcclient.NewJSONRPCClient(laddr)
	ctypes.RegisterAmino(client.Codec())
	result := new(ctypes.ResultStatus)
	for {
		_, err := client.Call("status", map[string]interface{}{}, result)
		if err == nil {
			return
		} else {
			t.Log("error", err)
			time.Sleep(time.Millisecond)
		}
	}
}

func waitForGRPC(t testing.TB, c *cfg.Config) {
	t.Helper()
	client := core_grpc.StartGRPCClient(c.RPC.GRPCListenAddress)
	for {
		_, err := client.Ping(context.Background(), &core_grpc.RequestPing{})
		if err == nil {
			return
		}
	}
}

// Do not use rpctest.StartTendermint or NewTendermint as they rely on a global
// state and cannot be used more than once.
func newTendermint(t testing.TB, config *cfg.Config, app abci.Application, logger log.Logger) *nm.Node {
	t.Helper()
	pvKeyFile := config.PrivValidatorKeyFile()
	pvKeyStateFile := config.PrivValidatorStateFile()
	pv := privval.LoadOrGenFilePV(pvKeyFile, pvKeyStateFile)
	papp := proxy.NewLocalClientCreator(app)
	nodeKey, err := p2p.LoadOrGenNodeKey(config.NodeKeyFile())
	if err != nil {
		t.Fatalf("cannot load or generate a node key: %s", err)
	}
	node, err := nm.NewNode(config, pv, nodeKey, papp,
		nm.DefaultGenesisDocProviderFunc(config),
		nm.DefaultDBProvider,
		nm.DefaultMetricsProvider(config.Instrumentation),
		logger)
	if err != nil {
		t.Fatalf("cannot create a new node: %s", err)
	}
	return node
}

func randomAddr(t testing.TB) string {
	port, err := cmn.GetFreePort()
	if err != nil {
		t.Fatalf("cannot acquire a free port: %s", err)
	}
	return fmt.Sprintf("tcp://0.0.0.0:%d", port)
}
