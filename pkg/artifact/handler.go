package artifact

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/x"
)

const (
	PackageName              = "artifact"
	createArtifactCost int64 = 100
	deleteArtifactCost int64 = 100
)

// RegisterQuery registers buckets for querying.
func RegisterQuery(qr weave.QueryRouter) {
	NewBucket().Register("artifacts", qr)
}

// RegisterRoutes registers handlers for message processing.
func RegisterRoutes(r weave.Registry, auth x.Authenticator) {
	r = migration.SchemaMigratingRegistry(PackageName, r)
	bucket := NewBucket()
	r.Handle(&CreateArtifactMsg{}, &CreateArtifactHandler{auth: auth, b: bucket})
	r.Handle(&DeleteArtifactMsg{}, &DeleteArtifactHandler{auth: auth, b: bucket})
}

type CreateArtifactHandler struct {
	auth x.Authenticator
	b    *Bucket
}

// Check just verifies it is properly formed and returns the cost of executing it.
func (h CreateArtifactHandler) Check(ctx weave.Context, store weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	_, err := h.validate(ctx, store, tx)
	if err != nil {
		return nil, err
	}
	return &weave.CheckResult{GasAllocated: createArtifactCost}, nil
}

// Deliver persists the artifact data if all preconditions are met
func (h CreateArtifactHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	msg, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, err
	}

	artifact := &Artifact{
		Metadata: &weave.Metadata{},
		Image:    msg.Image,
		Checksum: msg.Checksum,
		Owner:    msg.Owner,
	}
	if _, err := h.b.Put(db, []byte(msg.Image), artifact); err != nil {
		return nil, errors.Wrap(err, "failed to store artifact")
	}

	return &weave.DeliverResult{Data: []byte(msg.Image)}, err
}

// validate does all common pre-processing between Check and Deliver
func (h CreateArtifactHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*CreateArtifactMsg, error) {
	var msg CreateArtifactMsg

	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, errors.Wrap(err, "load msg")
	}

	if msg.Owner != nil {
		if !h.auth.HasAddress(ctx, msg.Owner) {
			return nil, errors.Wrap(errors.ErrUnauthorized, "owner's signature required")
		}
	} else {
		msg.Owner = x.MainSigner(ctx, h.auth).Address()
	}

	return &msg, nil
}

type DeleteArtifactHandler struct {
	auth x.Authenticator
	b    *Bucket
}

func (h DeleteArtifactHandler) Check(ctx weave.Context, store weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	_, err := h.validate(ctx, store, tx)
	if err != nil {
		return nil, err
	}
	return &weave.CheckResult{GasAllocated: deleteArtifactCost}, nil
}

func (h DeleteArtifactHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	msg, err := h.validate(ctx, db, tx)
	if err != nil {
		return nil, err
	}
	if err := h.b.Delete(db, []byte(msg.Image)); err != nil {
		return nil, errors.Wrap(err, "failed to delete entity")
	}
	return &weave.DeliverResult{}, err
}

func (h DeleteArtifactHandler) validate(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*DeleteArtifactMsg, error) {
	var msg DeleteArtifactMsg
	if err := weave.LoadMsg(tx, &msg); err != nil {
		return nil, errors.Wrap(err, "load msg")
	}

	var a Artifact
	if err := h.b.One(db, []byte(msg.Image), &a); err != nil {
		return nil, errors.Wrap(err, "cannot load artifact entity from the store")
	}
	if !h.auth.HasAddress(ctx, a.Owner) {
		return nil, errors.Wrap(errors.ErrUnauthorized, "owner's signature required")
	}

	return &msg, nil
}
