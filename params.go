package main

import (
	litecoinCfg "github.com/ltcsuite/ltcd/chaincfg"
	bitcoinCfg "github.com/roasbeef/btcd/chaincfg"
	"github.com/roasbeef/btcd/wire"
)

// activeNetParams is a pointer to the parameters specific to the currently
// active bitcoin network.
var activeNetParams = bitcoinTestNetParams

// bitcoinNetParams couples the p2p parameters of a network with the
// corresponding RPC port of a daemon running on the particular network.
type bitcoinNetParams struct {
	*bitcoinCfg.Params
	rpcPort string
}

// litecoinNetParams couples the p2p parameters of a network with the
// corresponding RPC port of a daemon running on the particular network.
type litecoinNetParams struct {
	*litecoinCfg.Params
	rpcPort string
}

// bitcoinTestNetParams contains parameters specific to the 3rd version of the
// test network.
var bitcoinTestNetParams = bitcoinNetParams{
	Params:  &bitcoinCfg.TestNet3Params,
	rpcPort: "18334",
}

// bitcoinSimNetParams contains parameters specific to the simulation test
// network.
var bitcoinSimNetParams = bitcoinNetParams{
	Params:  &bitcoinCfg.SimNetParams,
	rpcPort: "18556",
}

// liteTestNetParams contains parameters specific to the 4th version of the
// test network.
var liteTestNetParams = litecoinNetParams{
	Params:  &litecoinCfg.TestNet4Params,
	rpcPort: "19334",
}

// regTestNetParams contains parameters specific to a local regtest network.
var regTestNetParams = bitcoinNetParams{
	Params:  &bitcoinCfg.RegressionNetParams,
	rpcPort: "18334",
}

// liteSimNetParams contains parameters specific to the simulation
// test network.
var liteSimNetParams = litecoinNetParams{
	Params:  &litecoinCfg.SimNetParams,
	// it seems like ltcd listen on 18556 in simnet mode
	rpcPort: "18556",
}

// applyLitecoinParams applies the relevant chain configuration parameters that
// differ for litecoin to the chain parameters typed for btcsuite derivation.
// This function is used in place of using something like interface{} to
// abstract over _which_ chain (or fork) the parameters are for.
func applyLitecoinParams(params *bitcoinNetParams, liteNetParams *litecoinNetParams) {
	params.Name = liteNetParams.Name
	params.Net = wire.BitcoinNet(liteNetParams.Net)
	params.DefaultPort = liteNetParams.DefaultPort
	params.CoinbaseMaturity = liteNetParams.CoinbaseMaturity

	copy(params.GenesisHash[:], liteNetParams.GenesisHash[:])

	// Address encoding magics
	params.PubKeyHashAddrID = liteNetParams.PubKeyHashAddrID
	params.ScriptHashAddrID = liteNetParams.ScriptHashAddrID
	params.PrivateKeyID = liteNetParams.PrivateKeyID
	params.WitnessPubKeyHashAddrID = liteNetParams.WitnessPubKeyHashAddrID
	params.WitnessScriptHashAddrID = liteNetParams.WitnessScriptHashAddrID

	copy(params.HDPrivateKeyID[:], liteNetParams.HDPrivateKeyID[:])
	copy(params.HDPublicKeyID[:], liteNetParams.HDPublicKeyID[:])

	params.HDCoinType = liteNetParams.HDCoinType

	params.rpcPort = liteNetParams.rpcPort
}
