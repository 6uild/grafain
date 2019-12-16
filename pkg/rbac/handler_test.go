package rbac

import (
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/weavetest"
	"github.com/iov-one/weave/weavetest/assert"
)

func TestQueries(t *testing.T) {
	alice := weavetest.NewCondition().Address()

	db := store.MemStore()
	migration.MustInitPkg(db, PackageName)
	// given roles
	rBucket := NewRoleBucket()
	_, err := rBucket.Put(db, nil, &Role{
		Metadata: &weave.Metadata{Schema: 1},
		Address:  alice, // any
		Owner:    alice, // any
	})
	assert.Nil(t, err)

	// given principals
	pBucket := NewPrincipalBucket()
	_, err = pBucket.Put(db, nil, &Principal{
		Metadata: &weave.Metadata{Schema: 1},
		Signatures: []*NamedSignature{{
			Name:        "foo",
			Description: "test",
			Signature:   alice,
		}},
	})
	assert.Nil(t, err)

	// given bindings
	rBucket.Put(db, nil, &Role{})
	rbBucket := NewRoleBindingBucket()
	_, err = rbBucket.Create(db, weavetest.SequenceID(1), alice)
	assert.Nil(t, err)

	specs := map[string]struct {
		queryPath     string
		queryData     []byte
		expResultKeys [][]byte
	}{
		"find rolebinding by address": {
			queryPath:     "/rbac/rolebindings",
			queryData:     alice,
			expResultKeys: [][]byte{rbBucket.DBKey(append(alice, weavetest.SequenceID(1)...))},
		},
		"find role by id": {
			queryPath:     "/rbac/roles",
			queryData:     weavetest.SequenceID(1),
			expResultKeys: [][]byte{append([]byte(roleBucketName+":"), weavetest.SequenceID(1)...)},
		},
		"find principal by id": {
			queryPath:     "/rbac/principals",
			queryData:     weavetest.SequenceID(1),
			expResultKeys: [][]byte{append([]byte(principalBucketName+":"), weavetest.SequenceID(1)...)},
		},
		"find principal by address": {
			queryPath:     "/rbac/principals/signature",
			queryData:     alice,
			expResultKeys: [][]byte{append([]byte(principalBucketName+":"), weavetest.SequenceID(1)...)},
		},
		"find rolebinding by unknown address": {
			queryPath: "/rbac/rolebindings",
			queryData: []byte("unknown"),
		},
		"find role by unknown id": {
			queryPath: "/rbac/roles",
			queryData: weavetest.SequenceID(9999),
		},
		"find principal by unknown id": {
			queryPath: "/rbac/principals",
			queryData: weavetest.SequenceID(99999),
		},
		"find principal by unknown address": {
			queryPath: "/rbac/principals/signature",
			queryData: []byte("unknown"),
		},
	}
	for msg, spec := range specs {
		t.Run(msg, func(t *testing.T) {
			qr := weave.NewQueryRouter()
			RegisterQuery(qr)
			h := qr.Handler(spec.queryPath)
			if h == nil {
				t.Fatalf("expected handler for path %q but got nil", spec.queryPath)
			}
			m, err := h.Query(db, weave.PrefixQueryMod, spec.queryData)
			assert.Nil(t, err)
			assert.Equal(t, len(spec.expResultKeys), len(m))
			for i, v := range spec.expResultKeys {
				assert.Equal(t, v, m[i].Key)
			}
		})
	}
}
