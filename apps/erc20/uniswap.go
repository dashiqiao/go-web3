package erc20

import (
	"bytes"
	"github.com/dashiqiao/go-web3"
	"github.com/dashiqiao/go-web3/eth"
	"github.com/dashiqiao/go-web3/types"
	"github.com/ethereum/go-ethereum/common"
	eTypes "github.com/ethereum/go-ethereum/core/types"
	"math/big"
)

type UniSwap struct {
	contr         *eth.Contract
	w3            *web3.Web3
	confirmation  int
	txPollTimeout int
}

func NewUniSwap(w3 *web3.Web3, contractAddress common.Address) (*UniSwap, error) {
	contr, err := w3.Eth.NewContract(Uniswap_ABI, contractAddress.String())
	if err != nil {
		return nil, err
	}
	e := &UniSwap{
		contr:         contr,
		w3:            w3,
		txPollTimeout: 720,
	}
	return e, nil
}

func (e *UniSwap) SwapETHForExactTokens(amountOut *big.Int, path []common.Address, to common.Address, deadline int64, gasPrice *big.Int) (hash common.Hash, ng *big.Int, err error) {

	code, err := e.contr.EncodeABI("swapETHForExactTokens", amountOut, path, to, deadline)
	if err != nil {
		return common.Hash{}, big.NewInt(0), err
	}

	return e.invokeAndWait(code, gasPrice, nil, nil)
}

func (e *UniSwap) invokeAndWait(code []byte, gasPrice, gasTipCap, gasFeeCap *big.Int) (common.Hash, *big.Int, error) {
	gasLimit, err := e.EstimateGasLimit(e.contr.Address(), code, nil, nil)
	if err != nil {
		return common.Hash{}, big.NewInt(0), err
	}

	var tx *eTypes.Receipt
	if gasPrice != nil {
		tx, err = e.SyncSendRawTransactionForTx(gasPrice, gasLimit, e.contr.Address(), code, nil)
	}

	if err != nil {
		return common.Hash{}, big.NewInt(0), err
	}

	if e.confirmation == 0 {
		return tx.TxHash, big.NewInt(int64(gasLimit)), nil
	}

	//if err := e.WaitBlock(uint64(e.confirmation)); err != nil {
	//	return common.Hash{}, big.NewInt(0), err
	//}

	return tx.TxHash, big.NewInt(int64(gasLimit)), nil
}

func (e *UniSwap) EstimateGasLimit(to common.Address, data []byte, gasPrice, wei *big.Int) (uint64, error) {
	call := &types.CallMsg{
		To:    to,
		Data:  data,
		Gas:   types.NewCallMsgBigInt(big.NewInt(types.MAX_GAS_LIMIT)),
		Value: types.NewCallMsgBigInt(wei),
	}
	if gasPrice != nil {
		call.GasPrice = types.NewCallMsgBigInt(gasPrice)
	}

	var emptyAddr common.Address
	from := e.w3.Eth.Address()
	if !bytes.Equal(emptyAddr[:], from[:]) {
		call.From = from
	}

	gasLimit, err := e.w3.Eth.EstimateGas(call)
	if err != nil {
		return 0, err
	}
	return gasLimit, nil
}

func (e *UniSwap) SyncSendRawTransactionForTx(
	gasPrice *big.Int, gasLimit uint64, to common.Address, data []byte, wei *big.Int,
) (*eTypes.Receipt, error) {
	hash, err := e.w3.Eth.SendRawTransaction(to, wei, gasLimit, gasPrice, data)
	if err != nil {
		return nil, err
	}

	ret := new(eTypes.Receipt)
	ret.TxHash = hash

	return ret, nil
}
