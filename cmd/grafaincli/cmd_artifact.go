package main

import (
	"flag"
	"fmt"
	"io"

	grafain "github.com/alpe/grafain/pkg/app"
	"github.com/alpe/grafain/pkg/artifact"
	"github.com/iov-one/weave"
)

func cmdCreateArtifact(input io.Reader, output io.Writer, args []string) error {
	fl := flag.NewFlagSet("", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), `
Store a new Artifact data entry.

		`)
		fl.PrintDefaults()
	}
	var (
		imageFl  = fl.String("image", "", "Container image url like 'gcr.io/projectID/imagename@sha256:123456'")
		digestFl = fl.String("digest", "", "Hash or checksum value of a binary, or Docker Registry 2.0 digest of a container.")
		ownerFl  = flAddress(fl, "owner", "", "Optional address that is set as owner. The owner must also sign the Tx")
	)
	if err := fl.Parse(args); err != nil {
		flagDie("failed to parse arguments")

	}
	if len(*imageFl) == 0 {
		flagDie("image can not be empty")
	}
	if len(*digestFl) == 0 {
		flagDie("digest can not be empty")
	}
	tx := grafain.Tx{
		Sum: &grafain.Tx_CreateArtifactMsg{
			CreateArtifactMsg: &artifact.CreateArtifactMsg{
				Metadata: &weave.Metadata{Schema: 1},
				Owner:    *ownerFl,
				Image:    *imageFl,
				Checksum: *digestFl,
			},
		},
	}
	_, err := writeTx(output, &tx)
	return err
}

func cmdDeleteArtifact(input io.Reader, output io.Writer, args []string) error {
	fl := flag.NewFlagSet("", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), `
Delete an Artifact data entry.

		`)
		fl.PrintDefaults()
	}
	var (
		artifactIDFl = flSeq(fl, "id", "", "Id of the artifact to delete")
	)
	if err := fl.Parse(args); err != nil {
		flagDie("failed to parse arguments")

	}
	if len(*artifactIDFl) == 0 {
		flagDie("id can not be empty")
	}
	tx := grafain.Tx{
		Sum: &grafain.Tx_DeleteArtifactMsg{
			DeleteArtifactMsg: &artifact.DeleteArtifactMsg{
				Metadata: &weave.Metadata{Schema: 1},
				ID:       *artifactIDFl,
			},
		},
	}
	_, err := writeTx(output, &tx)
	return err

}
