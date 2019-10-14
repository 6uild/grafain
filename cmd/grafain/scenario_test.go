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
	"testing"
	"time"

	"github.com/alpe/grafain/cmd/grafain/testsupport"
	"github.com/alpe/grafain/pkg/webhook"
	"github.com/iov-one/weave"
	weaveclient "github.com/iov-one/weave/client"
	"github.com/iov-one/weave/commands/server"
	"github.com/iov-one/weave/crypto"
	"github.com/iov-one/weave/weavetest/assert"
	"github.com/iov-one/weave/x/cash"
	cmn "github.com/tendermint/tendermint/libs/common"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/rpc/client"
	tm "github.com/tendermint/tendermint/types"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	k8runtime "sigs.k8s.io/controller-runtime/pkg/runtime/scheme"
)

const TendermintLocalAddr = "localhost:26657"

var (
	tendermintAddress = flag.String("address", TendermintLocalAddr, "destination address of tendermint rpc")
	hexSeed           = flag.String("seed", "d34c1970ae90acf3405f2d99dcaca16d0c7db379f4beafcfdf667b9d69ce350d27f5fb440509dfa79ec883a0510bc9a9614c3d44188881f0c5e402898b4bf3c9", "private key seed in hex")
	delay             = flag.Duration("delay", 10*time.Millisecond, "duration to wait between test cases for rate limits")
	derivationPath    = flag.String("derivation", "", "bip44 derivation path: \"m/44'/234'/0'\"")
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
func TestEndToEndScenario(t *testing.T) {
	logger := log.NewTMLogger(log.NewSyncWriter(os.Stdout))
	logger = log.NewFilter(logger, log.AllowError())

	tmConf := buildTendermintConfig(t, "scenario")
	tmConf.Moniker = "ScenarioTest"
	aliceKey := parsePrivateKey(t)
	alice := aliceKey.PublicKey().Address()
	initGenesis(t, tmConf.GenesisFile(), alice)

	appGenFactory, storage := appWithStorage()
	abciApp, err := appGenFactory(&server.Options{
		Home:   tmConf.RootDir,
		Logger: logger,
		Debug:  true,
	})
	assert.Nil(t, err)

	hookPort, err := cmn.GetFreePort()
	assert.Nil(t, err)

	hookAddress := fmt.Sprintf("127.0.0.1:%d", hookPort)
	_, filename, _, _ := runtime.Caller(0)
	certDir := filepath.Join(filepath.Dir(filename), "../../contrib/pki")
	admissionPath := "/testing"

	node := newTendermint(t, tmConf, abciApp, logger.With("module", "tendermint"))
	assert.Nil(t, node.Start())
	defer node.Stop()

	go func() {
		cfg := config.GetConfigOrDie()

		gv := schema.GroupVersion{Group: "", Version: "v1"}
		s, err := (&k8runtime.Builder{GroupVersion: gv}).Register(&corev1.Pod{}, &corev1.PodList{}).Build()
		assert.Nil(t, err)

		opts := manager.Options{
			Scheme: s,
			MapperProvider: func(c *rest.Config) (meta.RESTMapper, error) {
				return testsupport.FakeMapper{}, nil
			},
		}

		mgr, err := manager.New(cfg, opts)
		assert.Nil(t, err)

		logger := node.Logger.With("module", "admission-hook")
		logger = log.NewFilter(logger, log.AllowDebug())

		assert.Nil(t, webhook.Start(mgr, hookAddress, certDir, admissionPath, storage, logger))

	}()
	waitForGRPC(t, tmConf)
	waitForRPC(t, tmConf)
	t.Log("Endpoints are up")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err = weaveclient.NewLocalClient(node).WaitForNextBlock(ctx)
	assert.Nil(t, err)

	client := testsupport.NewClient(client.NewLocal(node))
	// when create artifact
	tx := client.CreateArtifact(alice, "foo/bar:v0.0.1", "myChecksum")
	nonce := testsupport.NewNonce(client, alice)
	seq, err := nonce.Next()
	assert.Nil(t, err)
	err = testsupport.SignTx(tx, aliceKey, tmConf.ChainID(), seq)
	assert.Nil(t, err)
	rsp := client.BroadcastTxSync(tx, time.Second)
	assert.Nil(t, rsp.IsError())

	// then
	a, err := client.GetArtifactByID(rsp.Response.DeliverTx.Data)
	assert.Nil(t, err)
	assert.Equal(t, "myChecksum", a.Checksum)
	assert.Equal(t, alice, a.Owner)
	assert.Nil(t, err)

	all, err := client.ListArtifact()
	assert.Nil(t, err)

	if len(all) == 0 {
		t.Fatal("Expected non empty result")
	}
	// and when call to admission web hook with a known image
	hookClient := testsupport.NewAdmissionClient(t, certDir, hookAddress, admissionPath)
	content := podJson("foo/bar:v0.0.1")
	data := hookClient.Query(content)
	// then
	assert.Equal(t, true, data.Response.Allowed)
	assert.Equal(t, 200, data.Response.Status.Code)
	// and with an unknown image
	data = hookClient.Query(podJson("any/unknown:image"))
	// then
	assert.Equal(t, false, data.Response.Allowed)
	assert.Equal(t, 404, data.Response.Status.Code)
}

func initGenesis(t *testing.T, filename string, alice weave.Address) {
	t.Helper()

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
