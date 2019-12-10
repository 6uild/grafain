package rbac

import (
	"context"
	"testing"

	"github.com/iov-one/weave/weavetest/assert"
)

func TestAuthenticate_HasAddress(t *testing.T) {
	srcRoles := map[string]Role{
		"test": {Permissions: []Permission{"_test.authz"}},
	}
	ctx := withRBAC(context.TODO(), srcRoles)
	auth := &Authenticate{}
	assert.Equal(t, true, auth.HasAddress(ctx, RoleCondition([]byte("test")).Address()))
	assert.Equal(t, false, auth.HasAddress(ctx, RoleCondition([]byte("unknown")).Address()))
}
