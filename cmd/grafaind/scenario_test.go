package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/alpe/grafain/cmd/grafaind/testsupport"
	grafain "github.com/alpe/grafain/pkg/app"
	"github.com/alpe/grafain/pkg/client"
	"github.com/alpe/grafain/pkg/webhook"
	"github.com/iov-one/weave"
	weaveclient "github.com/iov-one/weave/client"
	"github.com/iov-one/weave/commands/server"
	"github.com/iov-one/weave/crypto"
	"github.com/iov-one/weave/weavetest"
	"github.com/iov-one/weave/weavetest/assert"
	"github.com/iov-one/weave/x/cash"
	"github.com/tendermint/tendermint/config"
	cmn "github.com/tendermint/tendermint/libs/common"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/node"
	rpcclient "github.com/tendermint/tendermint/rpc/client"
	tm "github.com/tendermint/tendermint/types"
)

const TendermintLocalAddr = "localhost:26657"

var (
	hexSeed = flag.String("seed", "d34c1970ae90acf3405f2d99dcaca16d0c7db379f4beafcfdf667b9d69ce350d27f5fb440509dfa79ec883a0510bc9a9614c3d44188881f0c5e402898b4bf3c9", "private key seed in hex")
)

func parsePrivateKey(t *testing.T) *crypto.PrivateKey {
	data, err := hex.DecodeString(*hexSeed)
	assert.Nil(t, err)
	assert.Equal(t, len(data), 64)
	return &crypto.PrivateKey{Priv: &crypto.PrivateKey_Ed25519{Ed25519: data}}
}

// TestEndToEndScenario covers a full round trip:
// * Create an artifact
// * Query it via abci
// * Query it via admission hook
// * Delete artifact
func TestEndToEndScenario(t *testing.T) {
	logger := log.NewTMLogger(log.NewSyncWriter(os.Stdout))
	logger = log.NewFilter(logger, log.AllowError())

	aliceKey := parsePrivateKey(t)
	alice := aliceKey.PublicKey().Address()

	// configure and start tendermint node with grafain abci
	tmConf := testsupport.BuildTendermintConfig(t, "scenario")
	tmConf.Moniker = "ScenarioTest"
	anyAddress := weave.Address(make([]byte, weave.AddressLength))

	initGenesis(t, tmConf.GenesisFile(), alice, anyAddress)

	abciApp, err := grafain.GenerateApp(&server.Options{
		Home:   tmConf.RootDir,
		Logger: logger,
		Debug:  true,
	})
	assert.Nil(t, err)

	node := testsupport.NewTendermint(t, tmConf, abciApp, logger.With("module", "tendermint"))
	assert.Nil(t, node.Start())
	defer node.Stop()
	chainID := tmConf.ChainID()

	hookRuntime := StartHook(t, node, tmConf)

	awaitTendermitUp(t, tmConf, node)
	awaitHookUp(t, hookRuntime)

	// now start testing grafain via abci operations
	gClient := client.NewClient(rpcclient.NewLocal(node))

	// create artifact should succeed
	adminGroupAddress, err := weave.ParseAddress("seq:rbac/role/2")
	assert.Nil(t, err)
	tx := gClient.CreateArtifact(adminGroupAddress, "foo/bar:v0.0.1", "myChecksum")
	nonce := client.NewNonce(gClient, alice)
	seq, err := nonce.Next()
	assert.Nil(t, err)
	err = client.SignTx(tx, aliceKey, chainID, seq)
	assert.Nil(t, err)
	rsp := gClient.BroadcastTxSync(tx, time.Second)
	assert.Nil(t, rsp.IsError())

	// then get and list artifact should succeed
	a, err := gClient.GetArtifactByImage(rsp.Response.DeliverTx.Data)
	assert.Nil(t, err)
	assert.Equal(t, "myChecksum", a.Checksum)
	assert.Equal(t, adminGroupAddress, a.Owner)
	assert.Nil(t, err)

	all, err := gClient.ListArtifact()
	assert.Nil(t, err)

	if len(all) == 0 {
		t.Fatal("Expected non empty result")
	}
	// and when call to admission web hook with a known image
	hookClient := testsupport.NewAdmissionClient(t, hookRuntime.CertDir, hookRuntime.HookAddress, hookRuntime.AdmissionPath)
	content := podJson("foo/bar:v0.0.1")
	data := hookClient.Query(content)
	assert.Equal(t, true, data.Response.Allowed)
	assert.Equal(t, 200, data.Response.Status.Code)
	// and a genesis image
	content = podJson("alpetest/grafain:vx.y.z")
	data = hookClient.Query(content)
	assert.Equal(t, true, data.Response.Allowed)
	assert.Equal(t, 200, data.Response.Status.Code)

	// and with an unknown image
	data = hookClient.Query(podJson("any/unknown:image"))
	assert.Equal(t, false, data.Response.Allowed)
	assert.Equal(t, 404, data.Response.Status.Code)

	// and when delete
	tx = gClient.DeleteArtifact("foo/bar:v0.0.1")
	seq, err = nonce.Next()
	assert.Nil(t, err)
	err = client.SignTx(tx, aliceKey, chainID, seq)
	assert.Nil(t, err)
	rsp = gClient.BroadcastTxSync(tx, time.Second)
	assert.Nil(t, rsp.IsError())
}

// Admin role contains a wildcard permission and is owned by a multiSig with Bert as member
// Bert does not have permissions himself.
func TestMultiSigAdminScenario(t *testing.T) {
	logger := log.NewTMLogger(log.NewSyncWriter(os.Stdout))
	logger = log.NewFilter(logger, log.AllowError())

	anyAddress := weave.Address(make([]byte, weave.AddressLength))

	bertKey := weavetest.NewKey()
	bert := bertKey.PublicKey().Address()

	// configure and start tendermint node with grafain abci
	tmConf := testsupport.BuildTendermintConfig(t, "scenario")
	tmConf.Moniker = "ScenarioTest"

	initGenesis(t, tmConf.GenesisFile(), anyAddress, bert)

	abciApp, err := grafain.GenerateApp(&server.Options{
		Home:   tmConf.RootDir,
		Logger: logger,
		Debug:  true,
	})
	assert.Nil(t, err)

	node := testsupport.NewTendermint(t, tmConf, abciApp, logger.With("module", "tendermint"))
	assert.Nil(t, node.Start())
	defer node.Stop()
	chainID := tmConf.ChainID()
	awaitTendermitUp(t, tmConf, node)

	gClient := client.NewClient(rpcclient.NewLocal(node))
	nonce := client.NewNonce(gClient, bert)
	seq, err := nonce.Next()
	assert.Nil(t, err)
	// when delete an image with signer authN
	tx := gClient.DeleteArtifact("alpetest/grafain:vx.y.z")
	err = client.SignTx(tx, bertKey, chainID, seq)
	assert.Nil(t, err)
	rsp := gClient.BroadcastTxSync(tx, time.Second)
	// then it should fail
	assert.Equal(t, true, strings.Contains(rsp.IsError().Error(), "insufficient permissions: unauthorized"))

	// and when TX contains multiSig ID
	tx = gClient.DeleteArtifact("alpetest/grafain:vx.y.z")
	client.AddMultiSig(tx, weavetest.SequenceID(1))
	err = client.SignTx(tx, bertKey, chainID, seq)
	assert.Nil(t, err)
	rsp = gClient.BroadcastTxSync(tx, time.Second)
	// then no errors as multiSig is has role
	assert.Nil(t, rsp.IsError())
}

type HookRuntime struct {
	CertDir, AdmissionPath, HookAddress string
	Abort                               chan error
}

func StartHook(t *testing.T, node *node.Node, tmConf *config.Config) HookRuntime {
	// configure and start admission web hook
	_, filename, _, _ := runtime.Caller(0)
	certDir := filepath.Join(filepath.Dir(filename), "../../contrib/pki")
	admissionPath := "/testing"
	hookAddress := localServerAddress(t)

	abort := make(chan error, 1)
	go func() { //
		logger := node.Logger.With("module", "admission-hook")
		logger = log.NewFilter(logger, log.AllowDebug())
		mgr, err := testsupport.LocalManager()
		if err != nil {
			abort <- err
			return
		}
		err = webhook.Start(mgr, tmConf.RPC.ListenAddress, hookAddress, certDir, admissionPath, logger)
		if err != nil {
			abort <- err
		}
	}()
	return HookRuntime{certDir, admissionPath, hookAddress, abort}
}

func awaitHookUp(t *testing.T, hookRuntime HookRuntime) {
	select {
	case err := <-hookRuntime.Abort:
		t.Fatalf("unexpected error: %+v", err)
	default: // when hook is up by now then it must be good
	}
}

func awaitTendermitUp(t *testing.T, tmConf *config.Config, node *node.Node) {
	// wait for tendermit up
	testsupport.WaitForGRPC(t, tmConf)
	testsupport.WaitForRPC(t, tmConf)
	t.Log("Endpoints are up")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err := weaveclient.NewLocalClient(node).WaitForNextBlock(ctx)
	assert.Nil(t, err)
}

func localServerAddress(t *testing.T) string {
	hookPort, err := cmn.GetFreePort()
	assert.Nil(t, err)
	hookAddress := fmt.Sprintf("127.0.0.1:%d", hookPort)
	return hookAddress
}

func initGenesis(t *testing.T, filename string, alice, bert weave.Address) {
	doc, err := tm.GenesisDocFromFile(filename)
	assert.Nil(t, err)
	doc.ConsensusParams.Block.TimeIotaMs = int64(1)
	type dict map[string]interface{}
	appState, err := json.MarshalIndent(dict{
		"conf": dict{
			"migration": dict{
				"admin": alice,
			},
			"cash": cash.Configuration{
				CollectorAddress: alice,
			},
			"msgfee": dict{
				"owner":     alice,
				"fee_admin": alice,
			},
		},
		"multisig": []dict{
			{
				"activation_threshold": 3,
				"admin_threshold":      3,
				"//name":               "admin multisig",
				"participants": []dict{
					{
						"signature": bert,
						"weight":    3,
					},
				},
			},
		},
		"rbac": dict{
			"roles": []dict{
				{
					"name":  "system.admin",
					"owner": "seq:rbac/role/1",
					"permissions": []string{
						"_grafain.*",
					},
				},
				{
					"name":  "k8.admin",
					"owner": "seq:rbac/role/1",
					"permissions": []string{
						"_grafain.artifacts.delete",
					},
				},
				{
					"name":     "k8s.devops",
					"owner":    "seq:rbac/role/1",
					"role_ids": []int{2},
					"permissions": []string{
						"_grafain.artifacts.create",
					},
				},
			},
			"principals": []dict{
				{
					"name": "Alice",
					"signatures": []dict{
						{
							"name":      "Signature test",
							"signature": alice,
						},
					},
				},
				{
					"name": "MultiSig managed by Bert",
					"signatures": []dict{
						{
							"name":      "MultiSig test",
							"signature": "cond:multisig/usage/0000000000000001",
						},
					},
				},
			},
			"role_bindings": []dict{
				{
					"role_id":   3,
					"signature": alice,
				},
				{
					"role_id":   1,
					"signature": "cond:multisig/usage/0000000000000001",
				},
			},
		},
		"artifacts": []dict{
			{
				"image":    "alpetest/grafain:vx.y.z",
				"owner":    "seq:rbac/role/1",
				"checksum": "anyValidChecksum",
			},
		},
		"initialize_schema": []dict{
			{"ver": 1, "pkg": "artifact"},
			{"ver": 1, "pkg": "batch"},
			{"ver": 1, "pkg": "cash"},
			{"ver": 1, "pkg": "cron"},
			{"ver": 1, "pkg": "currency"},
			{"ver": 1, "pkg": "distribution"},
			{"ver": 1, "pkg": "escrow"},
			{"ver": 1, "pkg": "gov"},
			{"ver": 1, "pkg": "msgfee"},
			{"ver": 1, "pkg": "multisig"},
			{"ver": 1, "pkg": "paychan"},
			{"ver": 1, "pkg": "rbac"},
			{"ver": 1, "pkg": "sigs"},
			{"ver": 1, "pkg": "utils"},
			{"ver": 1, "pkg": "validators"},
		},
	}, "", "  ")
	assert.Nil(t, err)
	doc.AppState = appState
	assert.Nil(t, doc.SaveAs(filename))
}

func podJson(image string) string {
	return fmt.Sprintf(`
{
  "kind": "AdmissionReview",
  "apiVersion": "admission.k8s.io/v1beta1",
  "request": {
    "uid": "181988ef-db4e-4023-9af8-ea1121ccfa9a",
    "kind": {
      "group": "",
      "version": "v1",
      "kind": "Pod"
    },
    "resource": {
      "group": "",
      "version": "v1",
      "resource": "pods"
    },
    "requestKind": {
      "group": "",
      "version": "v1",
      "kind": "Pod"
    },
    "requestResource": {
      "group": "",
      "version": "v1",
      "resource": "pods"
    },
    "name": "microbot5-85b6bcc585-zws9j",
    "namespace": "default",
    "operation": "CREATE",
    "userInfo": {
      "username": "system:serviceaccount:kube-system:replicaset-controller",
      "uid": "ce7d5264-51d2-4998-a1db-9d7cd751d167",
      "groups": [
        "system:serviceaccounts",
        "system:serviceaccounts:kube-system",
        "system:authenticated"
      ]
    },
    "object": {
      "kind": "Pod",
      "apiVersion": "v1",
      "metadata": {
        "name": "microbot5-85b6bcc585-zws9j",
        "generateName": "microbot5-85b6bcc585-",
        "namespace": "default",
        "uid": "bcc03889-33be-4390-b047-01d13cf4f51e",
        "creationTimestamp": "2019-10-13T12:14:13Z",
        "labels": {
          "app": "microbot5",
          "pod-template-hash": "85b6bcc585"
        },
        "ownerReferences": [
          {
            "apiVersion": "apps/v1",
            "kind": "ReplicaSet",
            "name": "microbot5-85b6bcc585",
            "uid": "1acfcf3c-2fee-4b31-a4f0-480f4d363ea8",
            "controller": true,
            "blockOwnerDeletion": true
          }
        ]
      },
      "spec": {
        "volumes": [
          {
            "name": "default-token-th7qf",
            "secret": {
              "secretName": "default-token-th7qf"
            }
          }
        ],
        "containers": [
          {
            "name": "microbot",
            "image": %q,
            "resources": {},
            "volumeMounts": [
              {
                "name": "default-token-th7qf",
                "readOnly": true,
                "mountPath": "/var/run/secrets/kubernetes.io/serviceaccount"
              }
            ],
            "terminationMessagePath": "/dev/termination-log",
            "terminationMessagePolicy": "File",
            "imagePullPolicy": "IfNotPresent"
          }
        ],
        "restartPolicy": "Always",
        "terminationGracePeriodSeconds": 30,
        "dnsPolicy": "ClusterFirst",
        "serviceAccountName": "default",
        "serviceAccount": "default",
        "securityContext": {},
        "schedulerName": "default-scheduler",
        "tolerations": [
          {
            "key": "node.kubernetes.io/not-ready",
            "operator": "Exists",
            "effect": "NoExecute",
            "tolerationSeconds": 300
          },
          {
            "key": "node.kubernetes.io/unreachable",
            "operator": "Exists",
            "effect": "NoExecute",
            "tolerationSeconds": 300
          }
        ],
        "priority": 0,
        "enableServiceLinks": true
      },
      "status": {
        "phase": "Pending",
        "qosClass": "BestEffort"
      }
    },
    "oldObject": null,
    "dryRun": false,
    "options": {
      "kind": "CreateOptions",
      "apiVersion": "meta.k8s.io/v1"
    }
  }
}
`, image)
}
