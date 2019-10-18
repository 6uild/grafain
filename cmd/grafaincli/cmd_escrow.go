package main

import (
	"flag"
	"fmt"
	"io"

	grafain "github.com/alpe/grafain/pkg/app"
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/x/escrow"
)

func cmdReleaseEscrow(input io.Reader, output io.Writer, args []string) error {
	fl := flag.NewFlagSet("", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), `
Create a transaction for releasing funds from given escrow.
		`)
		fl.PrintDefaults()
	}
	var (
		escrowFl = flSeq(fl, "escrow", "", "An ID of an escrow that is to be released.")
		amountFl = flCoin(fl, "amount", "", "Optional amount that is to be transferred from the escrow. The whole escrow hold amount is used if no value is provided.")
	)
	fl.Parse(args)

	var amount []*coin.Coin
	if !coin.IsEmpty(amountFl) {
		amount = append(amount, amountFl)
	}
	tx := &grafain.Tx{
		Sum: &grafain.Tx_EscrowReleaseMsg{
			EscrowReleaseMsg: &escrow.ReleaseMsg{
				Metadata: &weave.Metadata{Schema: 1},
				EscrowId: *escrowFl,
				Amount:   amount,
			},
		},
	}
	_, err := writeTx(output, tx)
	return err
}
