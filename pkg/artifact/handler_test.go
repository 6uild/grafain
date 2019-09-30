package artifact

import (
	"context"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/weavetest"
	"github.com/iov-one/weave/weavetest/assert"
)

func TestCreateArtifact(t *testing.T) {
	alice := weavetest.NewCondition()
	anyBody := weavetest.NewCondition()
	bucket := NewBucket()

	specs := map[string]struct {
		src           *CreateArtifactMsg
		expCheckErr   *errors.Error
		expDeliverErr *errors.Error
		expPersisted  func(t *testing.T, db weave.KVStore, res *weave.DeliverResult)
	}{
		"happy path": {
			src: &CreateArtifactMsg{
				Metadata: &weave.Metadata{Schema: 1},
				Image:    "example/image:version",
				Checksum: "anyValidChecksum",
				Owner:    alice.Address(),
			},
			expPersisted: func(t *testing.T, db weave.KVStore, res *weave.DeliverResult) {
				var l Artifact
				assert.Nil(t, bucket.One(db, res.Data, &l))
				assert.Equal(t, "example/image:version", l.Image)
				assert.Equal(t, "anyValidChecksum", l.Checksum)
				assert.Equal(t, alice.Address(), l.Owner)
			},
		},
		"main signer becomes owner when empty": {
			src: &CreateArtifactMsg{
				Metadata: &weave.Metadata{Schema: 1},
				Image:    "example/image:version",
				Checksum: "anyValidChecksum",
			},
			expPersisted: func(t *testing.T, db weave.KVStore, res *weave.DeliverResult) {
				var l Artifact
				assert.Nil(t, bucket.One(db, res.Data, &l))
				assert.Equal(t, "example/image:version", l.Image)
				assert.Equal(t, "anyValidChecksum", l.Checksum)
				assert.Equal(t, alice.Address(), l.Owner)
			},
		},
		"owner must sign on create": {
			src: &CreateArtifactMsg{
				Metadata: &weave.Metadata{Schema: 1},
				Image:    "example/image:version",
				Checksum: "anyValidChecksum",
				Owner:    anyBody.Address(),
			},
			expCheckErr:   errors.ErrUnauthorized,
			expDeliverErr: errors.ErrUnauthorized,
		},
		"empty image should be rejected": {
			src: &CreateArtifactMsg{
				Metadata: &weave.Metadata{Schema: 1},
				Image:    "",
				Checksum: "anyValidChecksum",
			},
			expCheckErr:   errors.ErrEmpty,
			expDeliverErr: errors.ErrEmpty,
		},
		"empty checksum should be rejected": {
			src: &CreateArtifactMsg{
				Metadata: &weave.Metadata{Schema: 1},
				Image:    "example/image:version",
				Checksum: "",
			},
			expCheckErr:   errors.ErrInput,
			expDeliverErr: errors.ErrInput,
		},
	}

	for msg, spec := range specs {
		t.Run(msg, func(t *testing.T) {
			db := store.MemStore()
			migration.MustInitPkg(db, PackageName)
			auth := &weavetest.Auth{Signers: []weave.Condition{alice}}

			r := CreateArtifactHandler{auth: auth, b: bucket}
			cache := db.CacheWrap()

			ctx := context.TODO()
			tx := &weavetest.Tx{Msg: spec.src}

			if _, err := r.Check(ctx, cache, tx); !spec.expCheckErr.Is(err) {
				t.Fatalf("check expected: %+v  but got %+v", spec.expCheckErr, err)
			}

			cache.Discard()

			res, err := r.Deliver(ctx, cache, tx)
			if !spec.expDeliverErr.Is(err) {
				t.Fatalf("check expected: %+v  but got %+v", spec.expDeliverErr, err)
			}

			if spec.expPersisted != nil {
				spec.expPersisted(t, cache, res)
			}
		})
	}
}

func TestDeleteArtifact(t *testing.T) {
	alice := weavetest.NewCondition()
	anyBody := weavetest.NewCondition()
	myExample := &Artifact{
		Metadata: &weave.Metadata{Schema: 1},
		Image:    "example/image:version",
		Checksum: "anyValidChecksum",
		Owner:    alice.Address(),
	}

	myArtifactID := weavetest.SequenceID(1)
	specs := map[string]struct {
		src           *DeleteArtifactMsg
		signer        weave.Condition
		expCheckErr   *errors.Error
		expDeliverErr *errors.Error
		expDeleted    bool
	}{
		"happy path": {
			src: &DeleteArtifactMsg{
				Metadata: &weave.Metadata{Schema: 1},
				ID:       myArtifactID,
			},
			signer:     alice,
			expDeleted: true,
		},
		"requires owner authz": {
			src: &DeleteArtifactMsg{
				Metadata: &weave.Metadata{Schema: 1},
				ID:       myArtifactID,
			},
			signer:        anyBody,
			expCheckErr:   errors.ErrUnauthorized,
			expDeliverErr: errors.ErrUnauthorized,
			expDeleted:    false,
		},
	}
	for msg, spec := range specs {
		t.Run(msg, func(t *testing.T) {
			db := store.MemStore()
			migration.MustInitPkg(db, PackageName)
			auth := &weavetest.Auth{Signers: []weave.Condition{spec.signer}}
			bucket := NewBucket()

			_, err := bucket.Put(db, myArtifactID, myExample)
			assert.Nil(t, err)

			r := DeleteArtifactHandler{auth: auth, b: bucket}
			cache := db.CacheWrap()

			ctx := context.TODO()
			tx := &weavetest.Tx{Msg: spec.src}

			if _, err := r.Check(ctx, cache, tx); !spec.expCheckErr.Is(err) {
				t.Fatalf("check expected: %+v  but got %+v", spec.expCheckErr, err)
			}

			cache.Discard()

			_, err = r.Deliver(ctx, cache, tx)
			if !spec.expDeliverErr.Is(err) {
				t.Fatalf("check expected: %+v  but got %+v", spec.expDeliverErr, err)
			}

			if spec.expDeleted {
				assert.IsErr(t, errors.ErrNotFound, bucket.One(cache, spec.src.ID, nil))
			}
		})
	}

}
