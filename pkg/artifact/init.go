package artifact

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
)

type Initializer struct{}

var _ weave.Initializer = (*Initializer)(nil)

// FromGenesis will parse initial artifacts data from genesis and save it to the database
func (Initializer) FromGenesis(opts weave.Options, params weave.GenesisParams, db weave.KVStore) error {
	type genesisArtifact struct {
		Image    Image         `json:"image"`
		Checksum string        `json:"checksum"`
		Owner    weave.Address `json:"owner"`
	}
	bucket := NewBucket()
	stream := opts.Stream("artifacts")
	for i := 0; ; i++ {
		var a genesisArtifact
		switch err := stream(&a); {
		case errors.ErrEmpty.Is(err):
			return nil
		case err != nil:
			return errors.Wrap(err, "cannot load username token")
		}

		newArtifact := Artifact{
			Metadata: &weave.Metadata{Schema: 1},
			Owner:    a.Owner,
			Image:    a.Image,
			Checksum: a.Checksum,
		}

		if err := newArtifact.Validate(); err != nil {
			return errors.Wrapf(err, "[%d] artifact %q is invalid", i, newArtifact.Image)
		}
		if err := bucket.Has(db, []byte(newArtifact.Image)); !errors.ErrNotFound.Is(err) {
			return errors.Wrapf(err, "[%d]  image must %q not exist", i, newArtifact.Image)
		}
		if _, err := bucket.Put(db, []byte(newArtifact.Image), &newArtifact); err != nil {
			return errors.Wrapf(err, "[%d] failed to store artifact %q", i, newArtifact.Image)
		}
	}
}
