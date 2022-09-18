package bep20

import (
	"bytes"
	"fmt"
	"math/big"
	"sync/atomic"
	"time"

	"github.com/dashiqiao/go-web3"
	"github.com/dashiqiao/go-web3/eth"
	"github.com/dashiqiao/go-web3/types"
	"github.com/ethereum/go-ethereum/common"
	eTypes "github.com/ethereum/go-ethereum/core/types"
)

type ERC20PancakeSwap struct {
	contr         *eth.Contract
	w3            *web3.Web3
	confirmation  int
	txPollTimeout int
}

func NewERC20PancakeSwap(w3 *web3.Web3, contractAddress common.Address) (*ERC20PancakeSwap, error) {
	contr, err := w3.Eth.NewContract(PancakeSwap_ABI, contractAddress.String())
	if err != nil {
		return nil, err
	}
	e := &ERC20PancakeSwap{
		contr:         contr,
		w3:            w3,
		txPollTimeout: 720,
	}
	return e, nil
}

func (e *ERC20PancakeSwap) Address() common.Address {
	return e.contr.Address()
}

func (e *ERC20PancakeSwap) SetConfirmation(blockCount int) {
	e.confirmation = blockCount
}

func (e *ERC20PancakeSwap) SetTxPollTimeout(txPollTimeout int) {
	e.txPollTimeout = txPollTimeout
}

func (e *ERC20PancakeSwap) Allowance(owner, spender common.Address) (*big.Int, error) {

	ret, err := e.contr.Call("allowance", owner, spender)
	if err != nil {
		return nil, err
	}

	allow, ok := ret.(*big.Int)
	if !ok {
		return nil, fmt.Errorf("invalid result %v, type %T", ret, ret)
	}
	return allow, nil
}

func (e *ERC20PancakeSwap) Decimals() (uint8, error) {
	ret, err := e.contr.Call("decimals")
	if err != nil {
		return 0, err
	}

	decimals, ok := ret.(uint8)
	if !ok {
		return 0, fmt.Errorf("invalid result %v, type %T", ret, ret)
	}
	return decimals, nil
}

func (e *ERC20PancakeSwap) Symbol() (string, error) {
	ret, err := e.contr.Call("symbol")
	if err != nil {
		return "", err
	}

	symbol, ok := ret.(string)
	if !ok {
		return "", fmt.Errorf("invalid result %v, type %T", ret, ret)
	}
	return symbol, nil
}

func (e *ERC20PancakeSwap) BalanceOf(owner common.Address) (*big.Int, error) {

	ret, err := e.contr.Call("balanceOf", owner)
	if err != nil {
		return nil, err
	}

	allow, ok := ret.(*big.Int)
	if !ok {
		return nil, fmt.Errorf("invalid result %v, type %T", ret, ret)
	}
	return allow, nil
}

func (e *ERC20PancakeSwap) Approve(spender common.Address, limit, gasPrice, gasTipCap, gasFeeCap *big.Int) (hash common.Hash, ng *big.Int, err error) {

	code, err := e.contr.EncodeABI("approve", spender, limit)
	if err != nil {
		return common.Hash{}, big.NewInt(0), err
	}

	hash, ng, err = e.invokeAndWait(code, big.NewInt(0), gasPrice, gasTipCap, gasFeeCap)
	return
}

func (e *ERC20PancakeSwap) Transfer(to common.Address, amount, gasPrice, gasTipCap, gasFeeCap *big.Int) (hash common.Hash, err error) {
	code, err := e.contr.EncodeABI("transfer", to, amount)
	if err != nil {
		return common.Hash{}, err
	}

	hash, _, err = e.invokeAndWait(code, big.NewInt(0), gasPrice, gasTipCap, gasFeeCap)
	return
}

func (e *ERC20PancakeSwap) SwapExactTokensForTokensSupportingFeeOnTransferTokens(amountIn, amountOutMin *big.Int, path []common.Address, to common.Address, deadline, gasPrice, gasLimit, gasTipCap, gasFeeCap *big.Int) (hash common.Hash, ng *big.Int, err error) {
	code, err := e.contr.EncodeABI("swapExactTokensForTokensSupportingFeeOnTransferTokens",
		amountIn, amountOutMin, path, to, deadline)
	if err != nil {
		return common.Hash{}, big.NewInt(0), err
	}

	//fmt.Println("")
	//fmt.Printf("code: === %x", code)
	//fmt.Println("")

	hash, ng, err = e.invokeAndWait(code, gasPrice, gasLimit, gasTipCap, gasFeeCap)
	//fmt.Println("ng == ", ng)
	return
}

func (e *ERC20PancakeSwap) SwapTokensForExactTokens(amountOut, amountInMax *big.Int, path []common.Address, to common.Address, deadline, gasPrice, gasLimit, gasTipCap, gasFeeCap *big.Int) (hash common.Hash, ng *big.Int, err error) {
	code, err := e.contr.EncodeABI("swapTokensForExactTokens",
		amountOut, amountInMax, path, to, deadline)
	if err != nil {
		return common.Hash{}, big.NewInt(0), err
	}
	hash, ng, err = e.invokeAndWait(code, gasPrice, gasLimit, gasTipCap, gasFeeCap)
	return
}

func (e *ERC20PancakeSwap) SwapTokensForExactTokensCall(amountOut, amountInMax *big.Int, path []common.Address, to common.Address, deadline, gasPrice, gasLimit, gasTipCap, gasFeeCap *big.Int) (hash common.Hash, err error) {
	code, err := e.contr.EncodeABI("swapTokensForExactTokens",
		amountOut, amountInMax, path, to, deadline)
	if err != nil {
		return common.Hash{}, err
	}
	hash, err = e.invokeAndWaitCall(code, gasPrice, gasLimit, gasTipCap, gasFeeCap, 0)
	return
}

func (e *ERC20PancakeSwap) SwapExactTokensForTokensSupportingFeeOnTransferTokensNonce(amountIn, amountOutMin *big.Int, path []common.Address, to common.Address, deadline, gasPrice, gasLimit, gasTipCap, gasFeeCap *big.Int, nonce uint64) (hash common.Hash, ng *big.Int, err error) {
	code, err := e.contr.EncodeABI("swapExactTokensForTokensSupportingFeeOnTransferTokens",
		amountIn, amountOutMin, path, to, deadline)
	if err != nil {
		return common.Hash{}, big.NewInt(0), err
	}
	hash, ng, err = e.invokeAndWaitNonce(code, gasPrice, gasLimit, gasTipCap, gasFeeCap, nonce)
	return
}

func (e *ERC20PancakeSwap) SwapExactTokensForTokensSupportingFeeOnTransferTokensCall(amountIn, amountOutMin *big.Int, path []common.Address, to common.Address, deadline, gasPrice, gasLimit, gasTipCap, gasFeeCap *big.Int) (common.Hash, error) {
	code, err := e.contr.EncodeABI("swapExactTokensForTokensSupportingFeeOnTransferTokens",
		amountIn, amountOutMin, path, to, deadline)
	if err != nil {
		return common.Hash{}, err
	}

	return e.invokeAndWaitCall(code, gasPrice, gasLimit, gasTipCap, gasFeeCap, 0)
}

func (e *ERC20PancakeSwap) SwapExactTokensForTokensSupportingFeeOnTransferTokensCallNonce(amountIn, amountOutMin *big.Int, path []common.Address, to common.Address, deadline, gasPrice, gasLimit, gasTipCap, gasFeeCap *big.Int, nonce uint64) (common.Hash, error) {
	code, err := e.contr.EncodeABI("swapExactTokensForTokensSupportingFeeOnTransferTokens",
		amountIn, amountOutMin, path, to, deadline)
	if err != nil {
		return common.Hash{}, err
	}

	return e.invokeAndWaitCall(code, gasPrice, gasLimit, gasTipCap, gasFeeCap, nonce)
}

func (e *ERC20PancakeSwap) EstimateGasLimit(to common.Address, data []byte, gasPrice, wei *big.Int) (uint64, error) {
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

func (e *ERC20PancakeSwap) WaitBlock(blockCount uint64) error {
	num, err := e.w3.Eth.GetBlockNumber()
	if err != nil {
		return err
	}
	ti := time.NewTicker(time.Second)
	defer ti.Stop()
	for {
		<-ti.C
		nextNum, err := e.w3.Eth.GetBlockNumber()
		if err != nil {
			return err
		}
		if nextNum >= num+blockCount {
			return nil
		}
	}
}

func (e *ERC20PancakeSwap) SyncSendRawTransactionForTx(
	gasPrice *big.Int, gasLimit uint64, to common.Address, data []byte, wei *big.Int,
) (*eTypes.Receipt, error) {
	hash, err := e.w3.Eth.SendRawTransaction(to, wei, gasLimit, gasPrice, data)
	if err != nil {
		return nil, err
	}

	ret := new(eTypes.Receipt)
	ret.TxHash = hash
	return ret, nil

	type ReceiptCh struct {
		ret *eTypes.Receipt
		err error
	}

	var timeoutFlag int32
	ch := make(chan *ReceiptCh, 1)

	go func() {
		for {
			receipt, err := e.w3.Eth.GetTransactionReceipt(hash)
			if err != nil && err.Error() != "not found" {
				ch <- &ReceiptCh{
					err: err,
				}
				break
			}
			if receipt != nil {
				ch <- &ReceiptCh{
					ret: receipt,
					err: nil,
				}
				break
			}
			if atomic.LoadInt32(&timeoutFlag) == 1 {
				break
			}
		}
		// fmt.Println("send tx done")
	}()

	select {
	case result := <-ch:
		if result.err != nil {
			return nil, err
		}

		return result.ret, nil
	case <-time.After(time.Duration(e.txPollTimeout) * time.Second):
		atomic.StoreInt32(&timeoutFlag, 1)
		return nil, fmt.Errorf("transaction was not mined within %v seconds, "+
			"please make sure your transaction was properly sent. Be aware that it might still be mined!", e.txPollTimeout)
	}
}

func (e *ERC20PancakeSwap) SyncSendRawTransactionForTxNonce(
	gasPrice *big.Int, gasLimit uint64, to common.Address, data []byte, wei *big.Int, nonce uint64,
) (*eTypes.Receipt, error) {
	hash, err := e.w3.Eth.SendRawTransactionByNonce(to, wei, gasLimit, gasPrice, data, nonce)
	if err != nil {
		return nil, err
	}

	ret := new(eTypes.Receipt)
	ret.TxHash = hash
	return ret, nil
	//type ReceiptCh struct {
	//	ret *eTypes.Receipt
	//	err error
	//}
	//
	//var timeoutFlag int32
	//ch := make(chan *ReceiptCh, 1)
	//
	//go func() {
	//	for {
	//		receipt, err := e.w3.Eth.GetTransactionReceipt(hash)
	//		if err != nil && err.Error() != "not found" {
	//			ch <- &ReceiptCh{
	//				err: err,
	//			}
	//			break
	//		}
	//		if receipt != nil {
	//			ch <- &ReceiptCh{
	//				ret: receipt,
	//				err: nil,
	//			}
	//			break
	//		}
	//		if atomic.LoadInt32(&timeoutFlag) == 1 {
	//			break
	//		}
	//	}
	//	// fmt.Println("send tx done")
	//}()
	//
	//select {
	//case result := <-ch:
	//	if result.err != nil {
	//		return nil, err
	//	}
	//
	//	return result.ret, nil
	//case <-time.After(time.Duration(e.txPollTimeout) * time.Second):
	//	atomic.StoreInt32(&timeoutFlag, 1)
	//	return nil, fmt.Errorf("transaction was not mined within %v seconds, "+
	//		"please make sure your transaction was properly sent. Be aware that it might still be mined!", e.txPollTimeout)
	//}
}

func (e *ERC20PancakeSwap) SyncSendEIP1559Tx(
	gasTipCap *big.Int,
	gasFeeCap *big.Int,
	gasLimit uint64,
	to common.Address,
	data []byte,
	wei *big.Int,
) (*eTypes.Receipt, error) {
	hash, err := e.w3.Eth.SendRawEIP1559Transaction(to, wei, gasLimit, gasTipCap, gasFeeCap, data)
	if err != nil {
		return nil, err
	}

	type ReceiptCh struct {
		ret *eTypes.Receipt
		err error
	}

	var timeoutFlag int32
	ch := make(chan *ReceiptCh, 1)

	go func() {
		for {
			receipt, err := e.w3.Eth.GetTransactionReceipt(hash)
			if err != nil && err.Error() != "not found" {
				ch <- &ReceiptCh{
					err: err,
				}
				break
			}
			if receipt != nil {
				ch <- &ReceiptCh{
					ret: receipt,
					err: nil,
				}
				break
			}
			if atomic.LoadInt32(&timeoutFlag) == 1 {
				break
			}
		}
		// fmt.Println("send tx done")
	}()

	select {
	case result := <-ch:
		if result.err != nil {
			return nil, err
		}

		return result.ret, nil
	case <-time.After(time.Duration(e.txPollTimeout) * time.Second):
		atomic.StoreInt32(&timeoutFlag, 1)
		return nil, fmt.Errorf("transaction was not mined within %v seconds, "+
			"please make sure your transaction was properly sent. Be aware that it might still be mined!", e.txPollTimeout)
	}
}

func (e *ERC20PancakeSwap) invokeAndWait(code []byte, gasPrice, gasLimit, gasTipCap, gasFeeCap *big.Int) (common.Hash, *big.Int, error) {
	estimateGasLimit, err := e.EstimateGasLimit(e.contr.Address(), code, nil, nil)
	if err != nil {
		return common.Hash{}, big.NewInt(0), err
	}
	estimateGasLimit += gasLimit.Uint64()
	//fmt.Println("estimateGasLimit : ", estimateGasLimit)
	var tx *eTypes.Receipt
	if gasPrice != nil {
		tx, err = e.SyncSendRawTransactionForTx(gasPrice, estimateGasLimit, e.contr.Address(), code, nil)
	} else {
		tx, err = e.SyncSendEIP1559Tx(gasTipCap, gasFeeCap, estimateGasLimit, e.contr.Address(), code, nil)
	}

	if err != nil {
		return common.Hash{}, big.NewInt(0), err
	}

	if e.confirmation == 0 {
		return tx.TxHash, big.NewInt(int64(estimateGasLimit)), nil
	}

	//if err := e.WaitBlock(uint64(e.confirmation)); err != nil {
	//	return common.Hash{}, big.NewInt(0), err
	//}

	return tx.TxHash, big.NewInt(int64(estimateGasLimit)), nil
}

func (e *ERC20PancakeSwap) invokeAndWaitNonce(code []byte, gasPrice, gasLimit, gasTipCap, gasFeeCap *big.Int, nonce uint64) (common.Hash, *big.Int, error) {
	estimateGasLimit, err := e.EstimateGasLimit(e.contr.Address(), code, nil, nil)
	if err != nil {
		return common.Hash{}, big.NewInt(0), err
	}
	estimateGasLimit += gasLimit.Uint64()
	//fmt.Println("estimateGasLimit : ", estimateGasLimit)
	var tx *eTypes.Receipt
	if gasPrice != nil {
		if nonce > 0 {
			tx, err = e.SyncSendRawTransactionForTxNonce(gasPrice, estimateGasLimit, e.contr.Address(), code, nil, nonce)
		} else {
			tx, err = e.SyncSendRawTransactionForTx(gasPrice, estimateGasLimit, e.contr.Address(), code, nil)
		}
	} else {
		tx, err = e.SyncSendEIP1559Tx(gasTipCap, gasFeeCap, estimateGasLimit, e.contr.Address(), code, nil)
	}

	if err != nil {
		return common.Hash{}, big.NewInt(0), err
	}

	if e.confirmation == 0 {
		return tx.TxHash, big.NewInt(int64(estimateGasLimit)), nil
	}

	//if err := e.WaitBlock(uint64(e.confirmation)); err != nil {
	//	return common.Hash{}, big.NewInt(0), err
	//}

	return tx.TxHash, big.NewInt(int64(estimateGasLimit)), nil
}

func (e *ERC20PancakeSwap) invokeAndWaitCall(code []byte, gasPrice, gasLimit, gasTipCap, gasFeeCap *big.Int, nonce uint64) (common.Hash, error) {
	var tx *eTypes.Receipt
	var err error
	if gasPrice != nil {
		if nonce > 0 {
			tx, err = e.SyncSendRawTransactionForTxNonce(gasPrice, gasLimit.Uint64(), e.contr.Address(), code, nil, nonce)
		} else {
			tx, err = e.SyncSendRawTransactionForTx(gasPrice, gasLimit.Uint64(), e.contr.Address(), code, nil)
		}
	} else {
		tx, err = e.SyncSendEIP1559Tx(gasTipCap, gasFeeCap, gasLimit.Uint64(), e.contr.Address(), code, nil)
	}

	if err != nil {
		return common.Hash{}, err
	}

	if e.confirmation == 0 {
		return tx.TxHash, nil
	}

	//if err := e.WaitBlock(uint64(e.confirmation)); err != nil {
	//	return common.Hash{}, err
	//}

	return tx.TxHash, nil
}
