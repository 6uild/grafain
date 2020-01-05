package artifact

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
)

func init() {
	migration.MustRegister(1, &CreateArtifactMsg{}, migration.NoModification)
	migration.MustRegister(1, &DeleteArtifactMsg{}, migration.NoModification)
}

var _ weave.Msg = (*CreateArtifactMsg)(nil)

// Path returns the routing path for this message.
func (CreateArtifactMsg) Path() string {
	return "artifacts/create"
}

func (m CreateArtifactMsg) Validate() error {
	var errs error
	errs = errors.AppendField(errs, "Metadata", m.Metadata.Validate())
	errs = errors.AppendField(errs, "Image", m.Image.Validate())
	if !isChecksum(m.Checksum) {
		errs = errors.AppendField(errs, "Checksum", errors.ErrInput)
	}
	errs = errors.AppendField(errs, "Owner", m.Owner.Validate())
	return errs
}

var _ weave.Msg = (*DeleteArtifactMsg)(nil)

// Path returns the routing path for this message.
func (DeleteArtifactMsg) Path() string {
	return "artifacts/delete"
}

func (m DeleteArtifactMsg) Validate() error {
	var errs error
	errs = errors.AppendField(errs, "Metadata", m.Metadata.Validate())
	return errors.AppendField(errs, "Image", m.Image.Validate())
}
