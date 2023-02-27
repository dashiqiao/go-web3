package erc20

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

type ERC20 struct {
	contr         *eth.Contract
	w3            *web3.Web3
	confirmation  int
	txPollTimeout int
}

func NewERC20(w3 *web3.Web3, contractAddress common.Address) (*ERC20, error) {
	contr, err := w3.Eth.NewContract(ERC20_ABI, contractAddress.String())
	if err != nil {
		return nil, err
	}
	e := &ERC20{
		contr:         contr,
		w3:            w3,
		txPollTimeout: 720,
	}
	return e, nil
}

func (e *ERC20) Address() common.Address {
	return e.contr.Address()
}

func (e *ERC20) SetConfirmation(blockCount int) {
	e.confirmation = blockCount
}

func (e *ERC20) SetTxPollTimeout(txPollTimeout int) {
	e.txPollTimeout = txPollTimeout
}

func (e *ERC20) Allowance(owner, spender common.Address) (*big.Int, error) {

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

func (e *ERC20) Decimals() (uint8, error) {
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

func (e *ERC20) Symbol() (string, error) {
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

func (e *ERC20) Name() (string, error) {
	ret, err := e.contr.Call("name")
	if err != nil {
		return "", err
	}

	symbol, ok := ret.(string)
	if !ok {
		return "", fmt.Errorf("invalid result %v, type %T", ret, ret)
	}
	return symbol, nil
}

func (e *ERC20) BalanceOf(owner common.Address) (*big.Int, error) {

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

func (e *ERC20) Approve(spender common.Address, limit, gasPrice, gasTipCap, gasFeeCap *big.Int) (hash common.Hash, ng *big.Int, err error) {

	code, err := e.contr.EncodeABI("approve", spender, limit)
	if err != nil {
		return common.Hash{}, big.NewInt(0), err
	}

	return e.invokeAndWait(code, gasPrice, gasTipCap, gasFeeCap)
}

func (e *ERC20) ApproveGasLimit(spender common.Address, limit, gasPrice, gasLimit, gasTipCap, gasFeeCap *big.Int) (hash common.Hash, ng *big.Int, err error) {

	code, err := e.contr.EncodeABI("approve", spender, limit)
	if err != nil {
		return common.Hash{}, big.NewInt(0), err
	}

	return e.invokeAndWaitGasLimit(code, gasPrice, gasLimit, gasTipCap, gasFeeCap)
}

func (e *ERC20) ApproveCall(spender common.Address, limit, gasPrice, gasLimit, gasTipCap, gasFeeCap *big.Int) (hash common.Hash, err error) {

	code, err := e.contr.EncodeABI("approve", spender, limit)
	if err != nil {
		return common.Hash{}, err
	}

	return e.invokeAndWaitCall(code, gasPrice, gasLimit, gasTipCap, gasFeeCap)
}

func (e *ERC20) Transfer(to common.Address, amount, gasPrice, gasTipCap, gasFeeCap *big.Int) (hash common.Hash, ng *big.Int, err error) {
	code, err := e.contr.EncodeABI("transfer", to, amount)
	if err != nil {
		return common.Hash{}, big.NewInt(0), err
	}
	return e.invokeAndWait(code, gasPrice, gasTipCap, gasFeeCap)
}

func (e *ERC20) TransferCall(to common.Address, amount, gasPrice, gasLimit, gasTipCap, gasFeeCap *big.Int) (hash common.Hash, err error) {
	code, err := e.contr.EncodeABI("transfer", to, amount)
	if err != nil {
		return common.Hash{}, err
	}
	return e.invokeAndWaitCall(code, gasPrice, gasLimit, gasTipCap, gasFeeCap)
}

func (e *ERC20) TransferNonce(to common.Address, amount, gasPrice, gasLimit, gasTipCap, gasFeeCap *big.Int, nonce uint64) (hash common.Hash, err error) {
	code, err := e.contr.EncodeABI("transfer", to, amount)
	if err != nil {
		return common.Hash{}, err
	}
	hash, _, err = e.invokeAndWaitNonce(code, gasPrice, gasLimit, gasTipCap, gasFeeCap, nonce)
	return
}

func (e *ERC20) EstimateGasLimit(to common.Address, data []byte, gasPrice, wei *big.Int) (uint64, error) {
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

func (e *ERC20) WaitBlock(blockCount uint64) error {
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

func (e *ERC20) SyncSendRawTransactionForTx(
	gasPrice *big.Int, gasLimit uint64, to common.Address, data []byte, wei *big.Int,
) (*eTypes.Receipt, error) {
	hash, err := e.w3.Eth.SendRawTransaction(to, wei, gasLimit, gasPrice, data)
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

func (e *ERC20) SyncSendEIP1559Tx(
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

func (e *ERC20) invokeAndWait(code []byte, gasPrice, gasTipCap, gasFeeCap *big.Int) (common.Hash, *big.Int, error) {
	gasLimit, err := e.EstimateGasLimit(e.contr.Address(), code, nil, nil)
	if err != nil {
		return common.Hash{}, big.NewInt(0), err
	}

	var tx *eTypes.Receipt
	if gasPrice != nil {
		tx, err = e.SyncSendRawTransactionForTx(gasPrice, gasLimit, e.contr.Address(), code, nil)
	} else {
		tx, err = e.SyncSendEIP1559Tx(gasTipCap, gasFeeCap, gasLimit, e.contr.Address(), code, nil)
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

func (e *ERC20) invokeAndWaitGasLimit(code []byte, gasPrice, newGasLimit, gasTipCap, gasFeeCap *big.Int) (common.Hash, *big.Int, error) {
	gasLimit, err := e.EstimateGasLimit(e.contr.Address(), code, nil, nil)
	if err != nil {
		return common.Hash{}, big.NewInt(0), err
	}
	gasLimit = gasLimit + newGasLimit.Uint64()
	var tx *eTypes.Receipt
	if gasPrice != nil {
		tx, err = e.SyncSendRawTransactionForTx(gasPrice, gasLimit, e.contr.Address(), code, nil)
	} else {
		tx, err = e.SyncSendEIP1559Tx(gasTipCap, gasFeeCap, gasLimit, e.contr.Address(), code, nil)
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

func (e *ERC20) invokeAndWaitCall(code []byte, gasPrice, gasLimit, gasTipCap, gasFeeCap *big.Int) (common.Hash, error) {

	var tx *eTypes.Receipt
	var err error
	if gasPrice != nil {
		tx, err = e.SyncSendRawTransactionForTx(gasPrice, gasLimit.Uint64(), e.contr.Address(), code, nil)
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

func (e *ERC20) invokeAndWaitNonce(code []byte, gasPrice, gasLimit, gasTipCap, gasFeeCap *big.Int, nonce uint64) (common.Hash, *big.Int, error) {
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

func (e *ERC20) SyncSendRawTransactionForTxNonce(
	gasPrice *big.Int, gasLimit uint64, to common.Address, data []byte, wei *big.Int, nonce uint64,
) (*eTypes.Receipt, error) {
	hash, err := e.w3.Eth.SendRawTransactionByNonce(to, wei, gasLimit, gasPrice, data, nonce)
	if err != nil {
		return nil, err
	}

	ret := new(eTypes.Receipt)
	ret.TxHash = hash
	return ret, nil
}
