package rbac

import (
	"encoding/binary"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
)

type Initializer struct{}

var _ weave.Initializer = (*Initializer)(nil)

type (
	genesisRole struct {
		Name    string        `json:"name"`
		RoleIDs []uint64      `json:"role_ids"`
		Owner   weave.Address `json:"owner"`
	}

	genesisRoleBinding struct {
		RoleID    uint64        `json:"role_id"`
		Signature weave.Address `json:"signature"`
	}

	genesisUser struct {
		Name       string          `json:"name"`
		Signatures []weave.Address `json:"signatures"`
	}

	GenesisRBAC struct {
		Roles        []genesisRole        `json:"roles"`
		Users        []genesisUser        `json:"users"`
		RoleBindings []genesisRoleBinding `json:"role_bindings"`
	}
)

// FromGenesis will parse initial artifacts data from genesis and save it to the database
func (Initializer) FromGenesis(opts weave.Options, params weave.GenesisParams, db weave.KVStore) error {

	var genesis GenesisRBAC
	if err := opts.ReadOptions("rbac", &genesis); err != nil {
		return err
	}
	if err := addRoles(db, genesis); err != nil {
		return err
	}

	if err := addUsers(db, genesis); err != nil {
		return err
	}
	if err := addRoleBindings(db, genesis); err != nil {
		return err
	}

	return nil
}

func addUsers(db weave.KVStore, genesis GenesisRBAC) error {
	bucket := NewUserBucket()
	for i, v := range genesis.Users {
		user := User{
			Metadata:  &weave.Metadata{Schema: 1},
			Name:      v.Name,
			Signature: v.Signatures,
		}
		if _, err := bucket.Put(db, nil, &user); err != nil {
			return errors.Wrapf(err, "cannot save #%d user", i)
		}
	}
	return nil
}

func addRoles(db weave.KVStore, genesis GenesisRBAC) error {
	bucket := NewRoleBucket()
	for i, v := range genesis.Roles {
		key, err := roleSeq.NextVal(db)
		if err != nil {
			return errors.Wrap(err, "cannot acquire ID")
		}
		roleIds := make([][]byte, len(v.RoleIDs))
		for j, id := range v.RoleIDs {
			idKey := encodeIDKey(id)
			if err := bucket.Has(db, idKey); errors.ErrNotFound.Is(err) {
				return errors.Wrapf(errors.ErrHuman, "Role dependency not exists: id %d required for %q", id, v.Name)
			}
			roleIds[j] = idKey
		}
		role := Role{
			Metadata: &weave.Metadata{Schema: 1},
			Address:  RoleCondition(key).Address(),
			RoleIds:  roleIds,
			Name:     v.Name,
			Owner:    v.Owner,
		}
		if _, err := bucket.Put(db, key, &role); err != nil {
			return errors.Wrapf(err, "cannot save #%d role", i)
		}
	}
	return nil
}

func addRoleBindings(db weave.KVStore, genesis GenesisRBAC) error {
	bucket := NewRoleBindingBucket()
	roleBucket := NewRoleBucket()
	userBucket := NewUserBucket()

	for i, v := range genesis.RoleBindings {
		roleIdKey := encodeIDKey(v.RoleID)
		rb := RoleBinding{
			Metadata:  &weave.Metadata{Schema: 1},
			RoleId:    roleIdKey,
			Signature: v.Signature,
		}
		if err := roleBucket.Has(db, roleIdKey); errors.ErrNotFound.Is(err) {
			return errors.Wrapf(errors.ErrHuman, "Role dependency not exists: id %d required for binding # %d", v.RoleID, i)
		}
		var xxx []User
		userIDs, err := userBucket.ByIndex(db, SignatureIndex, v.Signature, &xxx)
		if err != nil {
			return err
		}
		if len(userIDs) == 0 {
			return errors.Wrapf(errors.ErrHuman, "User dependency not exists: signature %q required for binding # %d", v.Signature.String(), i)

		}
		if _, err := bucket.Put(db, rb); err != nil {
			return errors.Wrapf(err, "cannot save #%d role-binding", i)
		}
	}
	return nil
}

func encodeIDKey(n uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, n)
	return b
}