# EIP1559-sender

EIP1559-sender is a Go-based Ethereum transaction sending tool specifically designed to support the EIP-1559 gas fee mechanism.

## Basic concepts
- ``maxFeePerGas >= (base Fee + priority Fee)``
- ``gas Fee = min(base Fee + priority Fee, maxFeePerGas) * gas limit``
- ``priority Fee = min(maxPriorityFeePerGas, maxFeePerGas - base Fee)``

## Features
- Utilizes the `go-ethereum` client library
- Implements EIP-1559's `DynamicFeeTx` transaction type
- Automatic estimation of `gasLimit`, `baseFee`, and `maxPriorityFeePerGas`
- Calculation of optimal `maxFeePerGas`
  
## Dependencies
- ``go get github.com/ethereum/go-ethereum``

## Install
``
go install github.com/gmh5225/EIP1559-sender@v0.0.2
``

## Uninstall
``
where/which EIP1559-sender
``

## Clean module cache
``
go clean -modcache
``

## Usage
```
eip1559_sender -privateKey ... -receiver 0x... -rpcURL https://... -chainID 421614 -tokenValue 0.1
```

## Example output
```
Connected to the RPC URL
Using specified chain ID: 421614
Sender's address: 0x059dC4EEe9328A9f333a7e813B2f5B4A52ADD4dF
Receiver address: 0xe091701aC9816D48248887147B41AE312d26e1C3
nonce: 31
Transfer amount: 0.001000 tokens (equivalent to 1000000000000000 Wei)
Base fee: 100000000
Suggested tip cap: 0
Max fee per gas: 200000000
Estimated gas limit: 25345
Transaction sent successfully! Transaction hash: 0x11a34e46dbc5c0af56e724b88ec12fbf041f0cc70b28a7de10bfd8433ea71c62
Please check the transaction status on the blockchain explorer
```


