package rbac

import (
	"encoding/hex"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/orm"
	"github.com/iov-one/weave/x"
)

const (
	roleParticipantGasCost = 10
)

type Decorator struct {
	auth          x.Authenticator
	roleBucket    orm.ModelBucket
	roleBinBucket *RoleBindingBucket
}

var _ weave.Decorator = Decorator{}

func NewDecorator(auth x.Authenticator) Decorator {
	return Decorator{
		auth:          auth,
		roleBucket:    NewRoleBucket(),
		roleBinBucket: NewRoleBindingBucket(),
	}
}

// Check enforces roles added to the context before calling down the stack
func (d Decorator) Check(ctx weave.Context, store weave.KVStore, tx weave.Tx, next weave.Checker) (*weave.CheckResult, error) {
	newCtx, cost, err := d.authRoles(ctx, store, tx)
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
func (d Decorator) Deliver(ctx weave.Context, store weave.KVStore, tx weave.Tx, next weave.Deliverer) (*weave.DeliverResult, error) {
	newCtx, _, err := d.authRoles(ctx, store, tx)
	if err != nil {
		return nil, err
	}

	return next.Deliver(newCtx, store, tx)
}

func (d Decorator) authRoles(ctx weave.Context, db weave.KVStore, tx weave.Tx) (weave.Context, int64, error) {
	roleIdsProcessed := make(map[string]struct{})
	for _, c := range d.auth.GetConditions(ctx) {
		roleIDs, err := d.roleBinBucket.FindRoleIDsByAddress(db, c.Address())
		if err != nil {
			return ctx, 0, err
		}
		println("found roleIDs: ", hex.EncodeToString(roleIDs[0]))
		for _, roleID := range roleIDs {
			err := d.loadRoles(db, roleID, roleIdsProcessed)
			if err != nil {
				return ctx, 0, err
			}
		}
	}
	var costs int64
	if len(roleIdsProcessed) != 0 {
		conds := make([]weave.Condition, 0, len(roleIdsProcessed))
		for id := range roleIdsProcessed {
			conds = append(conds, RoleCondition([]byte(id)))
			costs += roleParticipantGasCost
		}
		println("adding #conditions: ", len(conds))
		ctx = withRBAC(ctx, conds)
	}
	return ctx, costs, nil
}

func (d Decorator) loadRoles(db weave.KVStore, roleID []byte, roleIdsProcessed map[string]struct{}) error {
	if _, ok := roleIdsProcessed[string(roleID)]; ok {
		return nil
	}
	var r Role
	if err := d.roleBucket.One(db, roleID, &r); err != nil {
		println("failed to load role with id: ", hex.EncodeToString(roleID))
		return err
	}
	roleIdsProcessed[string(roleID)] = struct{}{}

	for _, id := range r.RoleIds {
		err := d.loadRoles(db, id, roleIdsProcessed)
		if err != nil {
			return err
		}
	}
	return nil
}
