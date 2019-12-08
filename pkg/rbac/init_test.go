package rbac

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
		"rbac": {
			"roles": [
				{ 
					"name": "first role",
					"owner": "seq:rbac/role/1"
				},
				{ 
					"name": "second role",
					"owner": "seq:rbac/role/1",
					"role_ids":[ 1 ]
				}
			],
			"principals": [
				{
					"name": "Anton",
					"signatures": [
						{
							"name": "first",
							"signature": "seq:test/anton/1"
						},
						{
							"name": "second",
							"signature": "seq:test/anton/2"
						}
					]
				},
				{
					"name": "Bert",
					"signatures": [
						{
							"name": "first",
							"signature": "seq:test/bert/1"
						},
						{
							"name": "second",
							"signature": "seq:test/bert/2"
						}
					]
				}
			],
			"role_bindings": [
				{
              	"role_id": 1,
				"signature": "seq:test/anton/1" 
				},
				{
              	"role_id": 2,
				"signature": "seq:test/anton/2" 
				},
				{
              	"role_id": 2,
				"signature": "seq:test/bert/2" 
				}
			]
		}
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

	b := NewRoleBucket()
	var first Role
	if err := b.One(db, weavetest.SequenceID(1), &first); err != nil {
		t.Fatalf("cannot get first role from the database: %s", err)
	}
	assert.Equal(t, RoleCondition(weavetest.SequenceID(1)).Address(), first.Owner)
	assert.Equal(t, 0, len(first.RoleIds))

	//assert.Equal(t, "anyValidChecksum", first.Checksum)
	var second Role
	if err := b.One(db, weavetest.SequenceID(2), &second); err != nil {
		t.Fatalf("cannot get second role from the database: %s", err)
	}
	assert.Equal(t, RoleCondition(weavetest.SequenceID(1)).Address(), second.Owner)
	assert.Equal(t, [][]byte{weavetest.SequenceID(1)}, second.RoleIds)

	u := NewPrincipalBucket()
	var anton Principal
	if err := u.One(db, weavetest.SequenceID(1), &anton); err != nil {
		t.Fatalf("cannot get first principal from the database: %s", err)
	}
	assert.Equal(t, "Anton", anton.Name)
	assert.Equal(t, 2, len(anton.Signatures))
	firstSig, _ := weave.ParseAddress("seq:test/anton/1")
	secondSig, _ := weave.ParseAddress("seq:test/anton/2")
	expSignatures := []*NamedSignature{{Name: "first", Signature: firstSig}, {Name: "second", Signature: secondSig}}
	assert.Equal(t, expSignatures, anton.Signatures)
	// todo
	var bert Principal
	if err := u.One(db, weavetest.SequenceID(2), &bert); err != nil {
		t.Fatalf("cannot get second principal from the database: %s", err)
	}
	assert.Equal(t, "Bert", bert.Name)
}
