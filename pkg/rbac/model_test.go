package rbac

import (
	"strings"
	"testing"

	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/weavetest/assert"
)

func TestPermissionAllows(t *testing.T) {
	specs := map[string]struct {
		src, other Permission
		exp        bool
	}{
		"same should be allowed": {
			src:   Permission("_test.foo"),
			other: Permission("_test.foo"),
			exp:   true,
		},
		"wildcard should cover same parent ": {
			src:   Permission("_test.foo.*"),
			other: Permission("_test.foo.bar"),
			exp:   true,
		},
		"wildcard should cover all with same parent ": {
			src:   Permission("_test.foo.*"),
			other: Permission("_test.foo.bar.other"),
			exp:   true,
		},
		"different should be rejected": {
			src:   Permission("_test.foo"),
			other: Permission("_test.bar"),
			exp:   false,
		},
		"wildcard should not cover parent ": {
			src:   Permission("_test.foo.*"),
			other: Permission("_test.foo"),
			exp:   false,
		},
		"wildcard should not cover different parent ": {
			src:   Permission("_test.foo.*"),
			other: Permission("_test.different.bar"),
			exp:   false,
		},
		"wildcard should not cover child of different parent ": {
			src:   Permission("_test.foo.*"),
			other: Permission("_test.different.bar.other"),
			exp:   false,
		},
		"wildcard must not be an argument": {
			src:   Permission("_test.foo.*"),
			other: Permission("_test.foo.*"),
			exp:   false,
		},
	}
	for msg, spec := range specs {
		t.Run(msg, func(t *testing.T) {
			if exp, got := spec.exp, spec.src.Allows(spec.other); exp != got {
				t.Errorf("expected %v but got %v", exp, got)
			}
		})
	}

}
func TestPermissionValidation(t *testing.T) {
	specs := map[string]struct {
		src    Permission
		expErr error
	}{
		"ends with a char": {
			src: Permission("_test.foo"),
		},
		"ends with a number": {
			src: Permission("_test.foo2"),
		},
		"wildcard ": {
			src: Permission("_test.foo.*"),
		},
		"min length ": {
			src: Permission("ab"),
		},
		"must not be too long": {
			src:    Permission(strings.Repeat("a", 129)),
			expErr: errors.ErrInput,
		},
		"must not be too short": {
			src:    Permission("a"),
			expErr: errors.ErrInput,
		},
		"must not end with a dot": {
			src:    Permission("_test.foo."),
			expErr: errors.ErrInput,
		},
		"must not end with a _": {
			src:    Permission("_test.foo_"),
			expErr: errors.ErrInput,
		},
		"must not end with a -": {
			src:    Permission("_test.foo.-"),
			expErr: errors.ErrInput,
		},
	}
	for msg, spec := range specs {
		t.Run(msg, func(t *testing.T) {
			assert.IsErr(t, spec.expErr, spec.src.Validate())
		})
	}

}
