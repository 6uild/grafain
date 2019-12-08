package rbac

import (
	"regexp"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/orm"
)

func RoleCondition(id []byte) weave.Condition {
	return weave.NewCondition("rbac", "role", id)
}

type Permission string

const maxPermissionLength = 128

var isPermission = regexp.MustCompile(`^[0-9a-z.\-_]{1,128}$`).MatchString

func (m Permission) Validate() error {
	switch l := len(m); {
	case l == 0:
		return errors.Field("permission", errors.ErrEmpty, "")
	case l > maxPermissionLength:
		return errors.Field("permission", errors.ErrInput, "must not exceed: %d chars", maxPermissionLength)
	case !isPermission(string(m)):
		return errors.Field("permission", errors.ErrInput, "invalid characters")
	}
	return nil
}

func (m *Role) Validate() error {
	return nil
}

func (m *Role) Copy() orm.CloneableData {
	return &Role{}
}

func (m *Principal) Validate() error {
	var errs error
	for _, v := range m.Signatures {
		if err := v.Validate(); err != nil {
			errs = errors.Append(errs, err)
		}
	}
	return errs
}

const maxNameLength = 64

func (m *NamedSignature) Validate() error {
	switch l := len(m.Name); {
	case l == 0:
		return errors.Field("name", errors.ErrEmpty, "")
	case l > maxNameLength:
		return errors.Field("name", errors.ErrInput, "must not exceed: %d chars", maxPermissionLength)
	}
	// todo: prevent empty name, description,
	// name, description, signature
	if err := m.Signature.Validate(); err != nil {
		return errors.Field("signature", err, "")
	}
	return nil
}

func (m *RoleBinding) Validate() error {
	return nil
}
