package rbac

import (
	"context"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/x"
)

type contextKey int // local to the rbac module

const (
	contextRBAC contextKey = iota
)

func withRBAC(ctx weave.Context, conds []weave.Condition) weave.Context {
	val, _ := ctx.Value(contextRBAC).([]weave.Condition)
	if val == nil {
		return context.WithValue(ctx, contextRBAC, conds)
	}

	return context.WithValue(ctx, contextRBAC, append(val, conds...))
}

// Authenticate gets/sets permissions on the given context key
type Authenticate struct {
}

var _ x.Authenticator = Authenticate{}

// GetConditions returns permissions previously set on this context
func (a Authenticate) GetConditions(ctx weave.Context) []weave.Condition {
	val, _ := ctx.Value(contextRBAC).([]weave.Condition)
	if val == nil {
		return nil
	}
	return val
}

// HasAddress returns true iff this address is in GetConditions
func (a Authenticate) HasAddress(ctx weave.Context, addr weave.Address) bool {
	for _, s := range a.GetConditions(ctx) {
		if addr.Equals(s.Address()) {
			return true
		}
	}
	return false
}
