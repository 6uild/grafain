package rbac

import (
	"github.com/alpe/grafain/pkg/orm"
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
	weaveORM "github.com/iov-one/weave/orm"
)

const roleBucketName = "role"
const userBucketName = "user"
const roleBindingBucketName = "rolebind"
const SignatureIndex = "signature"

const PackageName = "rbac"

var roleSeq = weaveORM.NewSequence("contracts", "id")

type RoleBucket struct {
	weaveORM.ModelBucket
}

func NewRoleBucket() *RoleBucket {
	b := orm.NewModelBucket(roleBucketName, &Role{},
		orm.WithIDSequence(roleSeq),
	)
	return &RoleBucket{
		ModelBucket: migration.NewModelBucket(PackageName, b),
	}
}

type UserBucket struct {
	weaveORM.ModelBucket
}

func NewUserBucket() *UserBucket {
	b := orm.NewModelBucket(userBucketName, &User{},
		orm.WithMultiIndex(SignatureIndex, indexSignatures, true),
	)
	return &UserBucket{
		ModelBucket: migration.NewModelBucket(PackageName, b),
	}
}

// indexSignature is an indexer implementation for user's signatures as a second index.
func indexSignatures(obj weaveORM.Object) (bytes [][]byte, e error) {
	if obj == nil {
		return nil, errors.Wrap(errors.ErrHuman, "cannot take index of nil")
	}
	v, ok := obj.Value().(*User)
	if !ok {
		return nil, errors.Wrap(errors.ErrHuman, "Can only take index of User")
	}
	r := make([][]byte, len(v.Signature))
	for i, v := range v.Signature {
		r[i] = v
	}
	return r, nil
}

type RoleBindingBucket struct {
	weaveORM.Bucket
}

func NewRoleBindingBucket() *RoleBindingBucket {
	bucket := migration.NewBucket(PackageName, roleBindingBucketName, &RoleBinding{})
	return &RoleBindingBucket{
		Bucket: bucket,
	}
}
func (b RoleBindingBucket) Put(db weave.KVStore, r RoleBinding) ([]byte, error) {
	key := buildKey(r)
	return key, b.Bucket.Save(db, weaveORM.NewSimpleObj(key, &r))
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

func buildKey(roleBinding RoleBinding) []byte {
	return append(roleBinding.Signature, roleBinding.RoleId...)
}
