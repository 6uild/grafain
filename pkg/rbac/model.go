package rbac

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/orm"
)

func RoleCondition(id []byte) weave.Condition {
	return weave.NewCondition("rbac", "role", id)
}

func (m *Role) Validate() error {
	return nil
}

func (m *Role) Copy() orm.CloneableData {
	return &Role{}
}

func (m *User) Validate() error {
	var errs error
	for _, v := range m.Signature {
		if err := v.Validate(); err != nil {
			errs = errors.Append(errs, err)
		}
	}
	return errs
}

func (m *User) Copy() orm.CloneableData {
	return &User{}
}

func (m *RoleBinding) Validate() error {
	return nil
}

func (m *RoleBinding) Copy() orm.CloneableData {
	return &RoleBinding{}
}
