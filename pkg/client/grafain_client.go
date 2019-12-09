package client

import (
	grafain "github.com/alpe/grafain/pkg/app"
	"github.com/alpe/grafain/pkg/artifact"
	"github.com/iov-one/weave"
	"github.com/iov-one/weave/crypto"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/x/sigs"
	"github.com/tendermint/tendermint/rpc/client"
)

type Client struct {
	*BaseClient
}

func NewClient(conn client.Client) *Client {
	return &Client{NewBaseClient(conn)}
}

func (c Client) CreateArtifact(owner weave.Address, image, checksum string) *grafain.Tx {
	tx := &grafain.Tx{
		Sum: &grafain.Tx_CreateArtifactMsg{
			CreateArtifactMsg: &artifact.CreateArtifactMsg{
				Metadata: &weave.Metadata{Schema: 1},
				Owner:    owner,
				Image:    artifact.Image(image),
				Checksum: checksum,
			},
		},
	}
	return tx
}
func (c Client) DeleteArtifact(image string) *grafain.Tx {
	tx := &grafain.Tx{
		Sum: &grafain.Tx_DeleteArtifactMsg{
			DeleteArtifactMsg: &artifact.DeleteArtifactMsg{
				Metadata: &weave.Metadata{Schema: 1},
				Image:    artifact.Image(image),
			},
		},
	}
	return tx
}

// SignTx modifies the tx in-place, adding signatures
func SignTx(tx *grafain.Tx, signer *crypto.PrivateKey, chainID string, nonce int64) error {
	sig, err := sigs.SignTx(signer, tx, chainID, nonce)
	if err != nil {
		return err
	}
	tx.Signatures = append(tx.Signatures, sig)
	return nil
}

func (c Client) GetArtifactsByChecksum(checksum string) ([]*artifact.Artifact, error) {
	resp, err := c.AbciQuery("/artifacts/checksum", []byte(checksum))
	if err != nil {
		return nil, err
	}
	if len(resp.Models) == 0 {
		return nil, errors.ErrNotFound
	}
	result := make([]*artifact.Artifact, len(resp.Models))
	for i, m := range resp.Models {
		var x artifact.Artifact
		if err := x.Unmarshal(m.Value); err != nil {
			return nil, err
		}
		result[i] = &x
	}
	return result, nil
}

func (c Client) GetArtifactByImage(id []byte) (*artifact.Artifact, error) {
	resp, err := c.AbciQuery("/artifacts", id)
	if err != nil {
		return nil, err
	}
	if len(resp.Models) == 0 {
		return nil, errors.ErrNotFound
	}
	var x artifact.Artifact
	return &x, x.Unmarshal(resp.Models[0].Value)
}

func (c Client) ListArtifact() ([]artifact.Artifact, error) {
	resp, err := c.AbciQuery("/artifacts"+"?"+weave.PrefixQueryMod, nil)
	if err != nil {
		return nil, err
	}

	out := make([]artifact.Artifact, len(resp.Models))
	for i, m := range resp.Models {
		var x artifact.Artifact
		if err := x.Unmarshal(m.Value); err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshall element: %d", i)
		}
		out[i] = x
	}
	return out, nil
}
