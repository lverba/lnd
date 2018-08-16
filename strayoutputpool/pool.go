package strayoutputpool

import (
	"github.com/btcsuite/btcd/blockchain"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"

	"github.com/lightningnetwork/lnd/lnwallet"
	"github.com/lightningnetwork/lnd/strayoutputpool/store"
)

// PoolServer is pool which contains a list of stray outputs that
// can be manually or automatically swept back into wallet.
type PoolServer struct {
	cfg   *PoolConfig
	store store.OutputStore
}

// NewPoolServer instantiate StrayOutputsPool with implementation
// of storing serialised outputs to database.
func NewPoolServer(config *PoolConfig) StrayOutputsPoolServer {
	return &PoolServer{
		cfg:   config,
		store: store.NewOutputDB(config.DB),
	}
}

// AddSpendableOutput adds spendable output to stray outputs pool.
func (d *PoolServer) AddSpendableOutput(
	output lnwallet.SpendableOutput) error {
	return d.store.AddStrayOutput(
		store.NewOutputEntity(output),
	)
}

// Sweep generates transaction for all added previously outputs to the wallet
// output address and broadcast it to the network.
func (d *PoolServer) Sweep() error {
	btx, err := d.GenSweepTx()
	if err != nil {
		return err
	}

	// Calculate base amount of transaction, needs only to show in
	// info log.
	var amount int64
	for _, txOut := range btx.MsgTx().TxOut {
		amount += txOut.Value
	}

	log.Infof("publishing sweep transaction for a set of stray inputs with amount: %v",
		amount)

	return d.cfg.PublishTransaction(btx.MsgTx())
}

// GenSweepTx fetches all stray outputs from database and
// generates sweep transaction for them.
func (d *PoolServer) GenSweepTx() (*btcutil.Tx, error) {
	// First, we obtain a new public key script from the wallet which we'll
	// sweep the funds to.
	pkScript, err := d.cfg.GenSweepScript()
	if err != nil {
		return nil, err
	}

	// Retrieve all stray outputs that can be swept back to the wallet,
	// for all of them we need to recalculate fee based on current fee
	// rate in time of triggering sweeping function.
	strayInputs, err := d.store.FetchAllStrayOutputs()
	if err != nil {
		return nil, err
	}

	return d.genSweepTx(pkScript, strayInputs...)
}

// genSweepTx generates sweep transaction for the list of stray outputs.
func (d *PoolServer) genSweepTx(pkScript []byte,
	strayOutputs ...store.OutputEntity) (*btcutil.Tx, error) {
	// Compute the total amount contained in all stored outputs
	// marked as strayed.
	var (
		totalAmt    btcutil.Amount
		txEstimator lnwallet.TxWeightEstimator
	)

	feePerKW, err := d.cfg.Estimator.EstimateFeePerKW(6)
	if err != nil {
		return nil, err
	}

	// With the fee calculated, we can now create the transaction using the
	// information gathered above and the provided retribution information.
	txn := wire.NewMsgTx(2)

	hashCache := txscript.NewTxSigHashes(txn)

	addWitness := func(idx int, so lnwallet.SpendableOutput) error {
		// Generate witness for this outpoint and transaction.
		witness, err := so.BuildWitness(d.cfg.Signer, txn, hashCache, idx)
		if err != nil {
			return err
		}

		txn.TxIn[idx].Witness = witness

		return nil
	}

	// Add standard output to our wallet.
	txEstimator.AddP2WKHOutput()

	for i, sOutput := range strayOutputs {
		txEstimator.AddWitnessInputByType(sOutput.Output().WitnessType())

		totalAmt += sOutput.Output().Amount()

		// Add spendable outputs to transaction.
		txn.AddTxIn(&wire.TxIn{
			PreviousOutPoint: *sOutput.Output().OutPoint(),
		})

		// Generate a witness for each output of the transaction.
		if err := addWitness(i, sOutput.Output()); err != nil {
			return nil, err
		}
	}

	txFee := feePerKW.FeeForWeight(int64(txEstimator.Weight()))

	txn.AddTxOut(&wire.TxOut{
		PkScript: pkScript,
		Value:    int64(totalAmt - txFee),
	})

	// Validate the transaction before signing
	btx := btcutil.NewTx(txn)
	if err := blockchain.CheckTransactionSanity(btx); err != nil {
		return nil, err
	}

	return btx, nil
}

// Start is launches checking of swept outputs by interval into database.
// It must be run as a goroutine.
func (d *PoolServer) Start() error {
	return nil
}

// Stop is launches checking of swept outputs by interval into database.
func (d *PoolServer) Stop() {

}
