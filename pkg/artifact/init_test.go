package artifact

import (
	"encoding/json"
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/store"
	"github.com/iov-one/weave/weavetest"
	"github.com/iov-one/weave/weavetest/assert"
)

func TestGenesisInitializer(t *testing.T) {
	const genesis = `
	{
		"artifacts": [
			{
				"image": "foo/bar:v1",
				"owner": "seq:test/alice/1",
                "checksum": "anyValidChecksum"

			},
			{
				"image": "prom/prometheus@sha256:97b61971c9bfd43337423c56d5209a288487eaecb05165bd1636176d381d9e4c",
				"owner": "seq:test/bob/1",
                "checksum": "sha256:97b61971c9bfd43337423c56d5209a288487eaecb05165bd1636176d381d9e4c"

			}
		]
	}
	`

	var opts weave.Options
	if err := json.Unmarshal([]byte(genesis), &opts); err != nil {
		t.Fatalf("cannot unmarshal genesis: %s", err)
	}

	db := store.MemStore()
	migration.MustInitPkg(db, PackageName)

	var ini Initializer
	if err := ini.FromGenesis(opts, weave.GenesisParams{}, db); err != nil {
		t.Fatalf("cannot load genesis: %s", err)
	}

	b := NewBucket()
	var first Artifact
	if err := b.One(db, []byte("foo/bar:v1"), &first); err != nil {
		t.Fatalf("cannot get first artifact from the database: %s", err)
	}
	assert.Equal(t, weave.NewCondition("test", "alice", weavetest.SequenceID(1)).Address(), first.Owner)
	assert.Equal(t, Image("foo/bar:v1"), first.Image)
	assert.Equal(t, "anyValidChecksum", first.Checksum)

	var second Artifact
	if err := b.One(db, []byte("prom/prometheus@sha256:97b61971c9bfd43337423c56d5209a288487eaecb05165bd1636176d381d9e4c"), &second); err != nil {
		t.Fatalf("cannot get second artifact from the database: %s", err)
	}
	assert.Equal(t, weave.NewCondition("test", "bob", weavetest.SequenceID(1)).Address(), second.Owner)
	assert.Equal(t, Image("prom/prometheus@sha256:97b61971c9bfd43337423c56d5209a288487eaecb05165bd1636176d381d9e4c"), second.Image)
	assert.Equal(t, "sha256:97b61971c9bfd43337423c56d5209a288487eaecb05165bd1636176d381d9e4c", second.Checksum)
}
