package artifact

import (
	"regexp"

	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/orm"
)

func init() {
	migration.MustRegister(1, &Artifact{}, migration.NoModification)
}

var isChecksum = regexp.MustCompile(`^[0-9a-zA-Z]{0,10}:?[0-9a-zA-Z]{1,100}$`).MatchString

var _ orm.Model = (*Artifact)(nil)

func (m *Artifact) Validate() error {
	var errs error
	errs = errors.AppendField(errs, "Metadata", m.Metadata.Validate())
	errs = errors.AppendField(errs, "owner", m.Owner.Validate())
	errs = errors.AppendField(errs, "Image", m.Image.Validate())

	if !isChecksum(m.Checksum) {
		errs = errors.AppendField(errs, "Checksum", errors.ErrInput)
	}
	return errs
}

func (m *Artifact) Copy() orm.CloneableData {
	return &Artifact{
		Metadata: m.Metadata.Copy(),
		Image:    m.Image,
		Checksum: m.Checksum,
	}
}

const maxImageLength = 255

type Image string

func (i Image) Validate() error {
	switch l := len(i); {
	case l == 0:
		return errors.ErrEmpty
	case l > maxImageLength:
		return errors.Wrapf(errors.ErrInput, "exceeds max length: %d", maxImageLength)
	}
	return nil
}
