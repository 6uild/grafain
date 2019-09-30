package xwebhook

import (
	"context"
	"os"
	"testing"

	grafain "github.com/alpe/grafain/cmd/grafain/app"
	"github.com/alpe/grafain/pkg/artifact"
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/app"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/store/iavl"
	"github.com/iov-one/weave/weavetest/assert"
	"github.com/tendermint/tendermint/libs/log"
	corev1 "k8s.io/api/core/v1"
)

func TestQueryWeave(t *testing.T) {
	memoryStore := iavl.MockCommitStore()
	db := memoryStore.CacheWrap()
	migration.MustInitPkg(db, artifact.PackageName)

	specs := map[string]struct {
		stored     []*artifact.Artifact
		queryImage string
		expError   *errors.Error
	}{
		"image exists": {
			stored: []*artifact.Artifact{{
				Metadata: &weave.Metadata{1},
				Image:    "foo/bar:1234",
				Checksum: "aValidChecksum",
			}},
			queryImage: "foo/bar:1234",
		},
		"image does not exist": {
			queryImage: "non/existing:1234",
			expError:   errors.ErrNotFound,
		},
	}
	for msg, spec := range specs {
		t.Run(msg, func(t *testing.T) {
			bucket := artifact.NewBucket()
			for _, v := range spec.stored {
				_, err := bucket.Put(db, nil, v)
				assert.Nil(t, err)
			}
			assert.Nil(t, db.Write())
			store := app.NewStoreApp("test-app", memoryStore, grafain.QueryRouter(coin.Coin{}), context.TODO())
			v := NewPodValidator(store, log.NewTMLogger(log.NewSyncWriter(os.Stdout)))
			// when
			err := v.doWithContainers([]corev1.Container{
				{
					Name:  "test",
					Image: spec.queryImage,
				},
			})
			// then
			assert.IsErr(t, spec.expError, err)
		})
	}
}
