package artifact

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/orm"
)

const checksumIndex = "checksum"
const bucketName = "artifact"

type Bucket struct {
	orm.ModelBucket
}

func NewBucket() *Bucket {
	b := orm.NewModelBucket(bucketName, &Artifact{},
		orm.WithIndex(checksumIndex, indexChecksum, false),
	)
	return &Bucket{
		ModelBucket: migration.NewModelBucket(PackageName, b),
	}
}

// Put saves given model in the database. Before inserting into
// database, model is validated using its Validate method.
// The key must be the artifact image and not empty.
// Using a key that already exists in the database cause the value to
// be overwritten.
func (b *Bucket) Put(db weave.KVStore, image []byte, m orm.Model) ([]byte, error) {
	if len(image) == 0 {
		return nil, errors.Wrap(errors.ErrInput, "empty key not allowed")
	}
	return b.ModelBucket.Put(db, image, m)
}

// indexChecksum is an indexer implementation for checksum as a second index.
func indexChecksum(obj orm.Object) (bytes []byte, e error) {
	if obj == nil {
		return nil, errors.Wrap(errors.ErrHuman, "cannot take index of nil")
	}
	v, ok := obj.Value().(*Artifact)
	if !ok {
		return nil, errors.Wrap(errors.ErrHuman, "Can only take index of Artifacts")
	}
	return []byte(v.Checksum), nil
}
