package rbac

import "github.com/iov-one/weave"

func RegisterQuery(qr weave.QueryRouter) {
	NewRoleBucket().Register("rbac/roles", qr)
	NewRoleBindingBucket().Register("rbac/rolebindings", qr)
	NewPrincipalBucket().Register("rbac/principals", qr)
}
