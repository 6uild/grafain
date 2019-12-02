package rbac

import (
	"testing"

	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/store"
)

func TestDecorator(t *testing.T) {
	db := store.MemStore()
	migration.MustInitPkg(db, PackageName)

}
