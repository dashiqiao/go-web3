package bep20

import (
	"bytes"
	"fmt"
	"github.com/dashiqiao/go-web3"
	"github.com/dashiqiao/go-web3/eth"
	"github.com/dashiqiao/go-web3/types"
	"github.com/ethereum/go-ethereum/common"
	eTypes "github.com/ethereum/go-ethereum/core/types"
	"math/big"
	"sync/atomic"
	"time"
)

type BEP20Factory struct {
	contr         *eth.Contract
	w3            *web3.Web3
	confirmation  int
	txPollTimeout int
}

func NewBEP20Factory(w3 *web3.Web3, contractAddress common.Address) (*BEP20Factory, error) {
	contr, err := w3.Eth.NewContract(Factory_ABI, contractAddress.String())
	if err != nil {
		return nil, err
	}
	e := &BEP20Factory{
		contr:         contr,
		w3:            w3,
		txPollTimeout: 720,
	}
	return e, nil
}

func (e *BEP20Factory) GetPair(address1, address2 common.Address, gasPrice *big.Int) (hash common.Address, err error) {
	//code, err := e.contr.EncodeABI("getPair", address1, address2)
	//if err != nil {
	//	return common.Hash{}, err
	//}
	//
	//hash, _, err = e.invokeAndWait(code, big.NewInt(0), gasPrice, nil, nil)

	ret, err := e.contr.Call("getPair", address1, address2)
	if err != nil {
		return common.HexToAddress(""), err
	}

	allow, ok := ret.(common.Address)
	if !ok {
		return common.HexToAddress(""), fmt.Errorf("invalid result %v, type %T", ret, ret)
	}
	return allow, nil
}

func (e *BEP20Factory) EstimateGasLimit(to common.Address, data []byte, gasPrice, wei *big.Int) (uint64, error) {
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

func (e *BEP20Factory) invokeAndWait(code []byte, gasPrice, gasLimit, gasTipCap, gasFeeCap *big.Int) (common.Hash, *big.Int, error) {
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

func (e *BEP20Factory) SyncSendRawTransactionForTx(
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

func (e *BEP20Factory) SyncSendEIP1559Tx(
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
