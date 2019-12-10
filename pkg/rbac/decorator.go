package rbac

import (
	"strings"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/x"
)

const (
	roleParticipantGasCost = 10
)

// AuthNDecorator handles authentication.
type AuthNDecorator struct {
	authN         x.Authenticator
	roleBucket    orm.ModelBucket
	roleBinBucket *RoleBindingBucket
}

func NewAuthNDecorator(auth x.Authenticator) AuthNDecorator {
	return AuthNDecorator{
		authN:         auth,
		roleBucket:    NewRoleBucket(),
		roleBinBucket: NewRoleBindingBucket(),
	}
}

// Check enforces roles added to the context before calling down the stack
func (d AuthNDecorator) Check(ctx weave.Context, store weave.KVStore, tx weave.Tx, next weave.Checker) (*weave.CheckResult, error) {
	newCtx, cost, err := d.authRoles(ctx, store)
	if err != nil {
		return nil, err
	}

	res, err := next.Check(newCtx, store, tx)
	if err != nil {
		return nil, err
	}
	res.GasPayment += cost
	return res, nil
}

// Deliver enforces roles added to the context before calling down the stack
func (d AuthNDecorator) Deliver(ctx weave.Context, store weave.KVStore, tx weave.Tx, next weave.Deliverer) (*weave.DeliverResult, error) {
	newCtx, _, err := d.authRoles(ctx, store)
	if err != nil {
		return nil, err
	}

	return next.Deliver(newCtx, store, tx)
}

func (d AuthNDecorator) authRoles(ctx weave.Context, db weave.KVStore) (weave.Context, int64, error) {
	var costs int64
	roleIdsProcessed := make(map[string]Role)
	for _, c := range d.authN.GetConditions(ctx) {
		roleIDs, err := d.roleBinBucket.FindRoleIDsByAddress(db, c.Address())
		if err != nil {
			return ctx, 0, err
		}
		for _, roleID := range roleIDs {
			err := d.loadRoles(db, roleID, roleIdsProcessed)
			if err != nil {
				return ctx, 0, err
			}
			costs += roleParticipantGasCost
		}
	}
	if len(roleIdsProcessed) != 0 {
		ctx = withRBAC(ctx, roleIdsProcessed)
	}
	return ctx, costs, nil
}

func (d AuthNDecorator) loadRoles(db weave.KVStore, roleID []byte, roleIdsProcessed map[string]Role) error {
	if _, ok := roleIdsProcessed[string(roleID)]; ok {
		return nil
	}
	var r Role
	if err := d.roleBucket.One(db, roleID, &r); err != nil {
		return err
	}
	roleIdsProcessed[string(roleID)] = r

	for _, id := range r.RoleIds {
		err := d.loadRoles(db, id, roleIdsProcessed)
		if err != nil {
			return err
		}
	}
	return nil
}

type Authorizator interface {
	HasPermission(ctx weave.Context, p Permission) bool
}

// AuthZDecorator handles authorization
type AuthZDecorator struct {
	authZ            Authorizator
	permissionPrefix string
}

func NewAuthZDecorator(auth Authorizator, permissionPrefix string) AuthZDecorator {
	return AuthZDecorator{
		authZ:            auth,
		permissionPrefix: permissionPrefix,
	}
}

// Check enforces permissions for the message stored in the context before calling down the stack
func (d AuthZDecorator) Check(ctx weave.Context, store weave.KVStore, tx weave.Tx, next weave.Checker) (*weave.CheckResult, error) {
	msg, err := tx.GetMsg()
	if err != nil {
		return nil, err
	}
	perm := d.resolvePermission(msg)
	if !d.authZ.HasPermission(ctx, perm) {
		return nil, errors.Wrap(errors.ErrUnauthorized, "insufficient permissions")
	}
	return next.Check(ctx, store, tx)
}

// Deliver enforces permissions for the message stored in the context before calling down the stack
func (d AuthZDecorator) Deliver(ctx weave.Context, store weave.KVStore, tx weave.Tx, next weave.Deliverer) (*weave.DeliverResult, error) {
	msg, err := tx.GetMsg()
	if err != nil {
		return nil, err
	}
	perm := d.resolvePermission(msg)

	if !d.authZ.HasPermission(ctx, perm) {
		return nil, errors.Wrap(errors.ErrUnauthorized, "insufficient permissions")
	}
	return next.Deliver(ctx, store, tx)
}

func (d AuthZDecorator) resolvePermission(msg weave.Msg) Permission {
	path := msg.Path()
	normalizedPath := strings.ToLower(strings.ReplaceAll(path, "/", "."))
	return Permission(strings.Join([]string{d.permissionPrefix, normalizedPath}, "."))
}
