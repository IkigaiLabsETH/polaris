package mempool

import (
	"fmt"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/skip-mev/pob/mempool"
	cosmlib "pkg.berachain.dev/polaris/cosmos/lib"
	"pkg.berachain.dev/polaris/cosmos/x/evm/types"
	"pkg.berachain.dev/polaris/eth/common"
	coretypes "pkg.berachain.dev/polaris/eth/core/types"
)

// validateAuctionTx returns true iff the ethereum transaction is a valid auction bid transaction. Since
// we do not have access to valid basic in the mempool, we must valid it here.
func (txConfig *Config) validateAuctionTx(ethTx *coretypes.Transaction) (bool, error) {
	// The user should not be sending any value to the builder contract
	if ethTx.Value().Cmp(sdk.ZeroInt().BigInt()) != 0 {
		return false, fmt.Errorf("a bid transaction must not send any %s to the builder contract", txConfig.evmDenom)
	}

	// The user should be sending valid bid info to the builder contract's bid function
	bidInfo, err := txConfig.getBidInfoFromEthTx(ethTx)
	if err != nil {
		return false, fmt.Errorf("transaction must be a valid bid transaction: %w", err)
	}

	// Since we do not have access to valid basic in the mempool, we must ensure that the bundle of
	// is valid here.
	if len(bidInfo.Transactions) == 0 {
		return false, fmt.Errorf("bundle of transactions must not be empty")
	}

	for _, tx := range bidInfo.Transactions {
		if len(tx) == 0 {
			return false, fmt.Errorf("transaction bundle must not contain empty transactions")
		}
	}

	return true, nil
}

// getBidInfoFromSdkTx returns the bid information from an Cosmos SDK transaction.
func (txConfig *Config) getBidInfoFromSdkTx(tx sdk.Tx) (*mempool.AuctionBidInfo, error) {
	ethTx, err := getEthTransactionRequest(tx)
	if err != nil {
		return nil, err
	}

	if ethTx == nil {
		return nil, fmt.Errorf("transaction is not an ethereum transaction")
	}

	return txConfig.getBidInfoFromEthTx(ethTx)
}

// getBidInfoFromEthTx returns the bid information from an ethereum transaction.
func (txConfig *Config) getBidInfoFromEthTx(ethTx *coretypes.Transaction) (*mempool.AuctionBidInfo, error) {
	data := ethTx.Data()
	if len(data) <= 4 {
		return nil, fmt.Errorf("transaction data is too short")
	}

	// Get the method name and the inputs from the transaction data
	methodSigData := data[:4]
	method, err := txConfig.contractABI.MethodById(methodSigData)
	if err != nil {
		return nil, err
	}

	// Get the inputs from the transaction data (bid, bundle, timeout)
	inputsSigData := data[4:]
	inputsMap := make(map[string]interface{})
	if err := method.Inputs.UnpackIntoMap(inputsMap, inputsSigData); err != nil {
		return nil, err
	}

	bid, ok := inputsMap["bid"].(*big.Int)
	if !ok {
		return nil, fmt.Errorf("invalid bid type: %T", inputsMap["bid"])
	}

	bundle, ok := inputsMap["transactions"].([][]byte)
	if !ok {
		return nil, fmt.Errorf("invalid bundle type: %T", inputsMap["bundle"])
	}

	timeout, ok := inputsMap["timeout"].(uint64)
	if !ok {
		return nil, fmt.Errorf("invalid timeout type: %T", inputsMap["timeout"])
	}

	from, err := getFromEthTx(ethTx)
	if err != nil {
		return nil, err
	}
	bidder := cosmlib.AddressToAccAddress(from)

	auctionBidInfo := &mempool.AuctionBidInfo{
		Transactions: bundle,
		Bid:          sdk.NewCoin(txConfig.evmDenom, sdk.NewIntFromBigInt(bid)),
		Bidder:       bidder,
		Timeout:      timeout,
	}

	return auctionBidInfo, nil
}

// getFromEthTx returns the sender of an Ethereum transaction.
func getFromEthTx(tx *coretypes.Transaction) (common.Address, error) {
	from, err := gethtypes.Sender(gethtypes.LatestSignerForChainID(tx.ChainId()), tx)
	return from, err
}

// getEthTransactionRequest returns the EthTransactionRequest message from a
// sdk transaction.
func getEthTransactionRequest(tx sdk.Tx) (*coretypes.Transaction, error) {
	msgEthTx := make([]*coretypes.Transaction, 0)
	for _, msg := range tx.GetMsgs() {
		if ethTxMsg, ok := msg.(*types.EthTransactionRequest); ok {
			msgEthTx = append(msgEthTx, ethTxMsg.AsTransaction())
		}
	}

	switch {
	case len(msgEthTx) == 0:
		return nil, nil
	case len(msgEthTx) == 1 && len(tx.GetMsgs()) == 1:
		return msgEthTx[0], nil
	default:
		return nil, fmt.Errorf("invalid transaction: %T", tx)
	}
}