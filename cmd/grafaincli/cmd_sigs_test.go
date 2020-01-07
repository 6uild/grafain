package main

import (
	"bytes"
	"io"
	"io/ioutil"
	"testing"

	grafain "github.com/alpe/grafain/pkg/app"
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/x/cash"
)

func TestCmdSignTransactionHappyPath(t *testing.T) {
	tx := &grafain.Tx{
		Sum: &grafain.Tx_CashSendMsg{
			CashSendMsg: &cash.SendMsg{
				Metadata: &weave.Metadata{Schema: 1},
			},
		},
	}
	var input bytes.Buffer
	if _, err := writeTx(&input, tx); err != nil {
		t.Fatalf("cannot marshal transaction: %s", err)
	}

	var output bytes.Buffer
	args := []string{
		"-tm", tmURL,
		"-key", mustCreateFile(t, bytes.NewReader(fromHex(t, privKeyHex))),
	}
	if err := cmdSignTransaction(&input, &output, args); err != nil {
		t.Fatalf("transaction signing failed: %s", err)
	}

	tx, _, err := readTx(&output)
	if err != nil {
		t.Fatalf("cannot read created transaction: %s", err)
	}

	if n := len(tx.Signatures); n != 1 {
		t.Fatalf("want one signature, got %d", n)
	}
}

func mustCreateFile(t testing.TB, r io.Reader) string {
	t.Helper()

	fd, err := ioutil.TempFile("", t.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer fd.Close()
	if _, err := io.Copy(fd, r); err != nil {
		t.Fatal(err)
	}
	if err := fd.Close(); err != nil {
		t.Fatal(err)
	}
	return fd.Name()
}
