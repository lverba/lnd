package contractcourt

import (
	"io"

	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"

	"github.com/lightningnetwork/lnd/lnwallet"
)

// ContractOutput implementation of SpendableOutput interface for
// contract resolvers.
type ContractOutput struct {
	preimage [32]byte

	lnwallet.BaseOutput
}

// NewContractOutput creates contract spendable output.
func NewContractOutput(
	amt btcutil.Amount,
	outpoint wire.OutPoint,
	witnessType lnwallet.WitnessType,
	signDesc lnwallet.SignDescriptor,
	preimage [32]byte,
) lnwallet.SpendableOutput {
	return &ContractOutput{
		preimage: preimage,
		BaseOutput: *lnwallet.NewBaseOutput(amt, outpoint,
			witnessType, signDesc),
	}
}

// NewDecodedContractOutput creates contract spendable output from
// serialized stream.
func NewDecodedContractOutput(r io.Reader) (lnwallet.SpendableOutput, error) {
	output := &ContractOutput{}

	return output, output.Decode(r)
}

// BuildWitness generate witness script for current spendable output.
func (s *ContractOutput) BuildWitness(signer lnwallet.Signer, txn *wire.MsgTx,
	hashCache *txscript.TxSigHashes, txinIdx int) ([][]byte, error) {

	switch s.WitnessType() {

	// Generates witness function for htlc success transaction
	case lnwallet.HtlcAcceptedRemoteSuccess:
		s.SignDesc().SigHashes = hashCache

		return lnwallet.SenderHtlcSpendRedeem(signer, s.SignDesc(), txn,
			s.preimage[:])

	default:
		return s.BaseOutput.BuildWitness(signer, txn, hashCache, txinIdx)
	}
}

// Encode serializes data of spendable output to serial data
func (s *ContractOutput) Encode(w io.Writer) error {
	if err := s.BaseOutput.Encode(w); err != nil {
		return err
	}

	if _, err := w.Write(s.preimage[:]); err != nil {
		return err
	}

	return nil
}

// Decode deserializes data of spendable output from serial data
func (s *ContractOutput) Decode(r io.Reader) error {
	if err := s.BaseOutput.Decode(r); err != nil {
		return err
	}

	if _, err := io.ReadFull(r, s.preimage[:]); err != nil && err != io.EOF {
		return err
	}

	return nil
}
