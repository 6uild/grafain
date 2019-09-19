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

var isChecksum = regexp.MustCompile(`^[0-9a-zA-Z]{16,64}$`).MatchString

var _ orm.Model = (*Artifact)(nil)

func (m *Artifact) Validate() error {
	var errs error
	errs = errors.AppendField(errs, "Metadata", m.Metadata.Validate())
	switch l := len(m.Image); {
	case l == 0:
		errs = errors.AppendField(errs, "Image", errors.ErrEmpty)
	case l > 255:
		errs = errors.AppendField(errs, "Image", errors.Wrap(errors.ErrInput, "too long"))
	}
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
