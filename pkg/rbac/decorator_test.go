package rbac

import (
	"context"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/weavetest"
	"github.com/iov-one/weave/weavetest/assert"
)

func TestAuthNDecorator(t *testing.T) {
	alice := weavetest.NewCondition()
	bert := weavetest.NewCondition()
	anyBody := weavetest.NewCondition()

	db := store.MemStore()
	migration.MustInitPkg(db, PackageName)

	myRole := Role{
		Metadata:    &weave.Metadata{Schema: 1},
		Name:        "my",
		Permissions: []Permission{"foo", "bar"},
	}
	myExtdRole := Role{
		Metadata:    &weave.Metadata{Schema: 1},
		Name:        "my extended Role",
		RoleIds:     [][]byte{weavetest.SequenceID(1)},
		Permissions: []Permission{"extended"},
	}

	roleBucket := NewRoleBucket()
	bindBucket := NewRoleBindingBucket()
	myRoleID, err := roleBucket.Put(db, nil, &myRole)
	assert.Nil(t, err)
	myExtRoleID, err := roleBucket.Put(db, nil, &myExtdRole)
	assert.Nil(t, err)

	_, err = bindBucket.Create(db, myRoleID, alice.Address())
	_, err = bindBucket.Create(db, myExtRoleID, bert.Address())
	assert.Nil(t, err)

	specs := map[string]struct {
		signer         weave.Condition
		expCheckErr    *errors.Error
		expDeliverErr  *errors.Error
		expPermissions []Permission
		expConds       []weave.Condition
	}{
		"happy path with single role": {
			signer:         alice,
			expPermissions: []Permission{"foo", "bar"},
			expConds:       []weave.Condition{RoleCondition(myRoleID)},
		},
		"happy path with embedded role": {
			signer:         bert,
			expPermissions: []Permission{"extended", "foo", "bar"},
			expConds:       []weave.Condition{RoleCondition(myExtRoleID), RoleCondition(myRoleID)},
		},
		"no role": {
			signer: anyBody,
		},
	}

	cache := db.CacheWrap()

	for msg, spec := range specs {
		t.Run(msg, func(t *testing.T) {

			auth := &weavetest.Auth{Signers: []weave.Condition{spec.signer}}
			decorator := NewAuthNDecorator(auth)

			anyTx := &weavetest.Tx{}
			var hn mockHandler
			stack := weavetest.Decorate(&hn, decorator)

			if _, err := stack.Check(context.TODO(), cache, anyTx); !spec.expCheckErr.Is(err) {
				t.Fatalf("check expected: %+v  but got %+v", spec.expCheckErr, err)
			}
			if spec.expCheckErr == nil {
				assert.Equal(t, 1, hn.CheckCallCount())
				// and verify all role conditions are set
				verifyContext(t, hn.ctx, spec.expPermissions, spec.expConds)
			}

			cache.Discard()
			hn.ctx = nil
			if _, err := stack.Deliver(context.TODO(), cache, anyTx); !spec.expDeliverErr.Is(err) {
				t.Fatalf("check expected: %+v  but got %+v", spec.expDeliverErr, err)
			}
			if spec.expDeliverErr != nil {
				return
			}
			assert.Equal(t, 1, hn.DeliverCallCount())
			// and verify all role conditions are set
			verifyContext(t, hn.ctx, spec.expPermissions, spec.expConds)
		})
	}
}

func verifyContext(t *testing.T, ctx weave.Context, expPermissions []Permission, expConds []weave.Condition) {
	t.Helper()
	conds, _ := ctx.Value(contextRBACConditions).([]weave.Condition)
	assert.Equal(t, len(expConds), len(conds))
	for i, c := range expConds {
		assert.Equal(t, true, c.Equals(conds[i]))
	}
	// and verify permissions
	perms, _ := ctx.Value(contextRBACPermissions).(map[Permission]struct{})
	assert.Equal(t, len(expPermissions), len(perms))
	for _, exp := range expPermissions {
		if _, exists := perms[exp]; !exists{
			t.Fatalf("expected permission %q", exp)
		}
	}
}

type mockHandler struct {
	weavetest.Handler
	ctx weave.Context
}

func (h *mockHandler) Check(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.CheckResult, error) {
	h.ctx = ctx
	return h.Handler.Check(ctx, db, tx)
}

func (h *mockHandler) Deliver(ctx weave.Context, db weave.KVStore, tx weave.Tx) (*weave.DeliverResult, error) {
	h.ctx = ctx
	return h.Handler.Deliver(ctx, db, tx)
}
