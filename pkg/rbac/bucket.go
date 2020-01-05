package rbac

import (
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/orm"
)

const roleBucketName = "role"
const principalBucketName = "principal"
const roleBindingBucketName = "rolebind"
const SignatureIndex = "signature"

const PackageName = "rbac"

var roleSeq = orm.NewSequence("role", "id")

type RoleBucket struct {
	orm.ModelBucket
}

func NewRoleBucket() *RoleBucket {
	b := orm.NewModelBucket(roleBucketName, &Role{},
		orm.WithIDSequence(roleSeq),
	)
	return &RoleBucket{
		ModelBucket: migration.NewModelBucket(PackageName, b),
	}
}

type PrincipalBucket struct {
	orm.ModelBucket
}

func NewPrincipalBucket() *PrincipalBucket {
	b := orm.NewModelBucket(principalBucketName, &Principal{},
		orm.WithIndex(SignatureIndex, indexSignatures, true),
	)
	return &PrincipalBucket{
		ModelBucket: migration.NewModelBucket(PackageName, b),
	}
}

// indexSignature is an indexer implementation for principal's signatures as a second index.
func indexSignatures(obj orm.Object) (bytes [][]byte, e error) {
	if obj == nil {
		return nil, errors.Wrap(errors.ErrHuman, "cannot take index of nil")
	}
	v, ok := obj.Value().(*Principal)
	if !ok {
		return nil, errors.Wrap(errors.ErrHuman, "Can only take index of Principal")
	}
	r := make([][]byte, len(v.Signatures))
	for i, v := range v.Signatures {
		r[i] = v.Signature
	}
	return r, nil
}

type RoleBindingBucket struct {
	orm.Bucket
}

func NewRoleBindingBucket() *RoleBindingBucket {
	bucket := migration.NewBucket(PackageName, roleBindingBucketName, &RoleBinding{})
	return &RoleBindingBucket{
		Bucket: bucket,
	}
}

func (b RoleBindingBucket) Create(db weave.KVStore, roleIdKey []byte, signature weave.Address) ([]byte, error) {
	r := RoleBinding{
		Metadata:  &weave.Metadata{Schema: 1},
		RoleId:    roleIdKey,
		Signature: signature,
	}
	key := buildKey(r)
	return key, b.Bucket.Save(db, orm.NewSimpleObj(key, &r))
}

func (b RoleBindingBucket) FindRoleIDsByAddress(db weave.KVStore, a weave.Address) ([][]byte, error) {
	models, err := b.Bucket.Query(db, weave.PrefixQueryMod, a)
	if err != nil {
		return nil, errors.Wrap(err, "can not query via prefix scan")
	}
	r := make([][]byte, len(models))
	prefixLength := len(b.DBKey([]byte{})) + weave.AddressLength
	for i, m := range models {
		r[i] = m.Key[prefixLength:]
	}
	return r, nil
}
func (mb *RoleBindingBucket) Register(name string, r weave.QueryRouter) {
	mb.Bucket.Register(name, r)
}

func buildKey(roleBinding RoleBinding) []byte {
	x := make([]byte, weave.AddressLength+8)
	copy(x, roleBinding.Signature)
	copy(x[weave.AddressLength:], roleBinding.RoleId)
	return x
}
