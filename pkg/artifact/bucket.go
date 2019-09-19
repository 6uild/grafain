package artifact

import (
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/orm"
)

type ArtifactBucket struct {
	orm.ModelBucket
}

const imageIndex = "image"
const bucketName = "artifact"

var artifactIDSeq = orm.NewSequence(bucketName, "id")

func NewBucket() *ArtifactBucket {
	b := orm.NewModelBucket(bucketName, &Artifact{}, orm.WithIDSequence(artifactIDSeq), orm.WithIndex(imageIndex, indexImage, true))
	return &ArtifactBucket{
		ModelBucket: migration.NewModelBucket(packageName, b),
	}
}

func indexImage(obj orm.Object) (bytes []byte, e error) {
	if obj == nil {
		return nil, errors.Wrap(errors.ErrHuman, "cannot take index of nil")
	}
	v, ok := obj.Value().(*Artifact)
	if !ok {
		return nil, errors.Wrap(errors.ErrHuman, "Can only take index of Artifacts")
	}
	return []byte(v.Image), nil
}
