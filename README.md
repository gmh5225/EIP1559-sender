# EIP1559-sender

EIP1559-sender is a Go-based Ethereum transaction sending tool specifically designed to support the EIP-1559 gas fee mechanism.

## Features
- Utilizes the `go-ethereum` client library
- Implements EIP-1559's `DynamicFeeTx` transaction type
- Automatic estimation of `gasLimit`, `baseFee`, and `gasTipCap`
- Calculation of optimal `maxFeePerGas`
  
## Dependencies
- ``go get github.com/ethereum/go-ethereum``

## Usage
```
eip1559_sender -privateKey ... -receiver 0x... -rpcURL https://... -chainID 1 -tokenValue 0.1
```
