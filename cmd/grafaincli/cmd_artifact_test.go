package main

import (
	"bytes"
	"testing"

	"github.com/alpe/grafain/pkg/artifact"
	"github.com/iov-one/weave/weavetest/assert"
)

func TestCreateArtifact(t *testing.T) {
	var output bytes.Buffer
	args := []string{
		"-image", "foo/bar:any",
		"-digest", "anyChecksum",
		"-owner", "b1ca7e78f74423ae01da3b51e676934d9105f282",
	}
	if err := cmdCreateArtifact(nil, &output, args); err != nil {
		t.Fatalf("cannot create a transaction: %s", err)
	}

	tx, _, err := readTx(&output)
	assert.Nil(t, err)

	txmsg, err := tx.GetMsg()
	assert.Nil(t, err)

	msg := txmsg.(*artifact.CreateArtifactMsg)

	assert.Equal(t, fromHex(t, "b1ca7e78f74423ae01da3b51e676934d9105f282"), msg.Owner)
	assert.Equal(t, "foo/bar:any", msg.Image)
	assert.Equal(t, "anyChecksum", msg.Checksum)
}

func TestDeleteArtifact(t *testing.T) {
	var output bytes.Buffer
	args := []string{
		"-image", "foo/bar:v0.0.1",
	}
	if err := cmdDeleteArtifact(nil, &output, args); err != nil {
		t.Fatalf("cannot create a transaction: %s", err)
	}

	tx, _, err := readTx(&output)
	assert.Nil(t, err)

	txmsg, err := tx.GetMsg()
	assert.Nil(t, err)

	msg := txmsg.(*artifact.DeleteArtifactMsg)
	assert.Equal(t, artifact.Image("foo/bar:v0.0.1"), msg.Image)
}
