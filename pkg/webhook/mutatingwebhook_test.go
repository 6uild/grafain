package webhook

import (
	"fmt"

	"github.com/alpe/grafain/pkg/artifact"
	grafain "github.com/alpe/grafain/pkg/client"
	"github.com/iov-one/weave"
)

//func TestQueryWeave(t *testing.T) {
//	db := store.MemStore()
//	migration.MustInitPkg(db, artifact.PackageName)
//
//	specs := map[string]struct {
//		stored     []*artifact.Artifact
//		queryImage string
//		expError   *errors.Error
//	}{
//		"image exists": {
//			stored: []*artifact.Artifact{{
//				Metadata: &weave.Metadata{Schema: 1},
//				Owner:    weavetest.NewCondition().Address(),
//				Image:    "foo/bar:1234",
//				Checksum: "aValidChecksum",
//			}},
//			queryImage: "foo/bar:1234",
//		},
//		"image does not exist": {
//			queryImage: "non/existing:1234",
//			expError:   errors.ErrNotFound,
//		},
//	}
//	for msg, spec := range specs {
//		t.Run(msg, func(t *testing.T) {
//			bucket := artifact.NewBucket()
//			for _, v := range spec.stored {
//				_, err := bucket.Put(db, []byte(v.Image), v)
//				assert.Nil(t, err)
//			}
//			store := newArtifactQueryAdapter(db)
//			v := NewMutatingWebHook(store, log.NewTMLogger(log.NewSyncWriter(os.Stdout)))
//			// when
//			err := v.doWithContainers([]corev1.Container{
//				{
//					Name:  "test",
//					Image: spec.queryImage,
//				},
//			})
//			// then
//			assert.IsErr(t, spec.expError, err)
//		})
//	}
//}

type artifactQueryAdapter struct {
	db     weave.KVStore
	router weave.QueryRouter
}

func newArtifactQueryAdapter(db weave.KVStore) *artifactQueryAdapter {
	router := weave.NewQueryRouter()
	artifact.NewBucket().Register("artifacts", router)
	return &artifactQueryAdapter{db: db, router: router}
}

func (q artifactQueryAdapter) AbciQuery(path string, data []byte) (grafain.AbciResponse, error) {
	handler := q.router.Handler(path)
	if handler == nil {
		return grafain.AbciResponse{}, fmt.Errorf("no handler for path %q", path)
	}
	m, err := handler.Query(q.db, weave.KeyQueryMod, data)
	return grafain.AbciResponse{Models: m, Height: 1}, err
}
