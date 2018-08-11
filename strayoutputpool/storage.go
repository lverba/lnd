package strayoutputpool

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/coreos/bbolt"
	"github.com/lightningnetwork/lnd/lnwallet"
	"github.com/roasbeef/btcd/txscript"
	"github.com/roasbeef/btcd/wire"
	"github.com/roasbeef/btcutil"
)

var (
	// strayOutputBucket is the name of the bucket within the database that
	// stores all data related to small outputs that should be combined to
	// minimise fee rate to proceed the transaction
	strayOutputBucket = []byte("stray-output")

	// ErrNoStrayOutputCreated is returned when bucket of stray outputs
	// hasn't been created.
	ErrNoStrayOutputCreated = fmt.Errorf("there are no existing stray outputs")

	byteOrder = binary.BigEndian
)

// spendableOutput basic implementation of SpendableOutput interface.
type
spendableOutput struct {
	amt         btcutil.Amount
	outpoint    wire.OutPoint
	witnessType lnwallet.WitnessType
	signDesc    lnwallet.SignDescriptor
}

// Amount returns amount of the output.
func (s *spendableOutput) Amount() btcutil.Amount {
	return s.amt
}

// OutPoint is previous transaction output.
func (s *spendableOutput) OutPoint() *wire.OutPoint {
	return &s.outpoint
}

// WitnessType returns types which attached to generation of witness data.
func (s *spendableOutput) WitnessType() lnwallet.WitnessType {
	return s.witnessType
}

// SignDesc is used to signing raw transaction.
func (s *spendableOutput) SignDesc() *lnwallet.SignDescriptor {
	return &s.signDesc
}

func (s *spendableOutput) BuildWitness(signer lnwallet.Signer, txn *wire.MsgTx,
	hashCache *txscript.TxSigHashes, txinIdx int) ([][]byte, error) {
	// Now that we have ensured that the witness generation function has
	// been initialized, we can proceed to execute it and generate the
	// witness for this particular breached output.
	witnessFunc := s.witnessType.GenWitnessFunc(
		signer, s.SignDesc(),
	)

	return witnessFunc(txn, hashCache, txinIdx)
}

// AddStrayOutput saves serialized stray output to database in order to combine
// them to one transaction to pay fee for one transaction.
func (d *DBStrayOutputsPool) AddStrayOutput(output *strayOutput) error {
	var b bytes.Buffer
	if err := output.Encode(&b); err != nil {
		return err
	}

	return d.cfg.DB.Batch(func(tx *bolt.Tx) error {
		outputs, err := tx.CreateBucketIfNotExists(strayOutputBucket)
		if err != nil {
			return err
		}

		outputID, err := outputs.NextSequence()
		if err != nil {
			return err
		}

		paymentIDBytes := make([]byte, 8)
		binary.BigEndian.PutUint64(paymentIDBytes, outputID)

		return outputs.Put(paymentIDBytes, b.Bytes())
	})
}

// FetchAllStrayOutputs returns all stray outputs in DB.
func (d *DBStrayOutputsPool) FetchAllStrayOutputs() ([]*strayOutput, error) {
	var outputs []*strayOutput
	err := d.cfg.DB.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(strayOutputBucket)
		if bucket == nil {
			return ErrNoStrayOutputCreated
		}

		return bucket.ForEach(func(k, v []byte) error {
			output := &strayOutput{}
			if err := output.Decode(bytes.NewReader(v)); err != nil {
				return err
			}

			outputs = append(outputs, output)

			return nil
		})
	})
	if err != nil {
		return nil, err
	}

	return outputs, nil
}

// strayOutput
type strayOutput struct {
	// txVSize
	txVSize int64

	// totalAmt
	totalAmt btcutil.Amount

	// outputs
	outputs []lnwallet.SpendableOutput
}


// Encode
func (s *strayOutput) Encode(w io.Writer) error {
	var scratch [8]byte

	byteOrder.PutUint64(scratch[:], uint64(s.txVSize))
	if _, err := w.Write(scratch[:]); err != nil {
		return err
	}

	byteOrder.PutUint64(scratch[:], uint64(s.totalAmt))
	if _, err := w.Write(scratch[:]); err != nil {
		return err
	}

	for _, input := range s.outputs {
		byteOrder.PutUint64(scratch[:], uint64(input.Amount()))
		if _, err := w.Write(scratch[:]); err != nil {
			return err
		}

		if err := writeOutpoint(w, input.OutPoint()); err != nil {
			return err
		}

		byteOrder.PutUint16(scratch[:2], uint16(input.WitnessType()))
		if _, err := w.Write(scratch[:2]); err != nil {
			return err
		}

		if err := lnwallet.WriteSignDescriptor(w, input.SignDesc()); err != nil {
			return err
		}
	}

	return nil
}

// Decode
func (s *strayOutput) Decode(r io.Reader) error {
	var scratch [8]byte

	if _, err := r.Read(scratch[:]); err != nil {
		return err
	}
	s.txVSize = int64(byteOrder.Uint64(scratch[:]))

	if _, err := r.Read(scratch[:]); err != nil {
		return err
	}
	s.totalAmt = btcutil.Amount(byteOrder.Uint64(scratch[:]))

	var err error
	for {
		sOutput := &spendableOutput{}

		if _, err = r.Read(scratch[:]); err != nil {
			break
		}
		sOutput.amt = btcutil.Amount(byteOrder.Uint64(scratch[:]))

		if err = readOutpoint(io.LimitReader(r, 40),
			&sOutput.outpoint); err != nil {
			break
		}

		if _, err = r.Read(scratch[:2]); err != nil {
			break
		}
		sOutput.witnessType = lnwallet.WitnessType(
			byteOrder.Uint16(scratch[:2]),
		)

		if err = lnwallet.ReadSignDescriptor(r, &sOutput.signDesc); err != nil {
			break
		}

		s.outputs = append(s.outputs, sOutput)
	}
	if err != nil && err != io.EOF {
		return err
	}

	return nil
}

// writeOutpoint writes an outpoint to the passed writer using the minimal
// amount of bytes possible.
func writeOutpoint(w io.Writer, o *wire.OutPoint) error {
	if _, err := w.Write(o.Hash[:]); err != nil {
		return err
	}
	if err := binary.Write(w, byteOrder, o.Index); err != nil {
		return err
	}

	return nil
}

// readOutpoint reads an outpoint from the passed reader that was previously
// written using the writeOutpoint struct.
func readOutpoint(r io.Reader, o *wire.OutPoint) error {
	if _, err := io.ReadFull(r, o.Hash[:]); err != nil {
		return err
	}
	if err := binary.Read(r, byteOrder, &o.Index); err != nil {
		return err
	}

	return nil
}