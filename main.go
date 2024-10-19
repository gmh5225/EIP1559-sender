package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/big"
	"os"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

func main() {
	privateKeyFlag := flag.String("privateKey", "", "Sender's private key")
	receiverFlag := flag.String("receiver", "", "Receiver's address")
	rpcURLFlag := flag.String("rpcURL", "", "RPC URL")
	chainIDFlag := flag.Int64("chainID", 0, "Chain ID (if 0, it will be automatically obtained)")
	tokenValueFlag := flag.Float64("tokenValue", 0, "Transfer amount")

	// Add usage information
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [options]\n", os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(), "\nOptions:\n")
		flag.PrintDefaults()
		fmt.Fprintf(flag.CommandLine.Output(), "\nExample:\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  %s -privateKey 0x... -receiver 0x... -rpcURL https://... -chainID 1 -tokenValue 0.1\n", os.Args[0])
	}

	flag.Parse()

	// Check if required parameters are provided
	if *privateKeyFlag == "" || *receiverFlag == "" || *rpcURLFlag == "" || *tokenValueFlag == 0 {
		fmt.Println("Error: Missing required parameters")
		flag.Usage()
		os.Exit(1)
	}

	// get private key and receiver address
	privateKeyAddress := *privateKeyFlag
	receiverAddress := *receiverFlag

	// connect to RPC URL
	client, err := ethclient.Dial(*rpcURLFlag)
	if err != nil {
		log.Fatalf("Failed to connect to the RPC URL: %v", err)
	}
	fmt.Printf("Connected to the RPC URL %s\n", *rpcURLFlag)

	// get chain id
	var chainID *big.Int
	if *chainIDFlag != 0 {
		chainID = big.NewInt(*chainIDFlag)
		fmt.Printf("Using specified chain ID: %d\n", chainID)
	} else {
		chainID, err = client.ChainID(context.Background())
		if err != nil {
			log.Fatalf("Failed to get chain ID: %v", err)
		}
		fmt.Printf("Automatically obtained chain ID: %d\n", chainID)
	}

	privateKey, err := crypto.HexToECDSA(privateKeyAddress)
	if err != nil {
		log.Fatalf("Failed to parse private key: %v", err)
	}

	// get sender's address
	fromAddress := crypto.PubkeyToAddress(privateKey.PublicKey)
	toAddress := common.HexToAddress(receiverAddress)
	fmt.Printf("Sender's address: %s\n", fromAddress.Hex())
	fmt.Printf("Receiver address: %s\n", toAddress.Hex())

	// get nonce
	nonce, err := client.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		log.Fatalf("Failed to get nonce: %v", err)
	}
	fmt.Println("nonce:", nonce)

	// set transfer amount
	tokenValue := *tokenValueFlag
	weiPerToken := big.NewInt(1e18)
	tokenValueBigFloat := new(big.Float).SetFloat64(tokenValue)
	weiValueBigInt, _ := new(big.Float).Mul(tokenValueBigFloat, new(big.Float).SetInt(weiPerToken)).Int(nil)
	fmt.Printf("Transfer amount: %.6f tokens (equivalent to %s Wei)\n", tokenValue, weiValueBigInt.String())

	// get base fee
	header, err := client.HeaderByNumber(context.Background(), nil)
	if err != nil {
		log.Fatalf("Failed to get header: %v", err)
	}
	baseFee := header.BaseFee
	fmt.Printf("Base fee: %s\n", baseFee.String())

	// get suggested tip cap (maxPriorityFeePerGas)
	maxPriorityFeePerGas, err := client.SuggestGasTipCap(context.Background())
	if err != nil {
		log.Fatalf("Failed to get suggested maxPriorityFeePerGas: %v", err)
	}
	fmt.Printf("Suggested maxPriorityFeePerGas: %s\n", maxPriorityFeePerGas.String())

	// calculate maxFeePerGas (usually baseFee * 2 + maxPriorityFeePerGas)
	maxFeePerGas := new(big.Int).Add(
		new(big.Int).Mul(baseFee, big.NewInt(2)),
		maxPriorityFeePerGas,
	)
	fmt.Printf("Max fee per gas: %s\n", maxFeePerGas.String())

	// estimate gas limit
	gasLimit, err := client.EstimateGas(context.Background(), ethereum.CallMsg{
		From:  fromAddress,
		To:    &toAddress,
		Value: weiValueBigInt,
	})
	if err != nil {
		log.Fatalf("Failed to estimate gas: %v", err)
	}
	fmt.Printf("Estimated gas limit: %d\n", gasLimit)

	// create EIP-1559 transaction
	tx := types.NewTx(&types.DynamicFeeTx{
		ChainID:   chainID,
		Nonce:     nonce,
		GasTipCap: maxPriorityFeePerGas,
		GasFeeCap: maxFeePerGas,
		Gas:       gasLimit,
		To:        &toAddress,
		Value:     weiValueBigInt,
		Data:      nil,
	})

	// sign transaction
	signedTx, err := types.SignTx(tx, types.LatestSignerForChainID(chainID), privateKey)
	if err != nil {
		log.Fatalf("Failed to sign transaction: %v", err)
	}

	// send transaction
	err = client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		log.Fatalf("Failed to send transaction: %v", err)
	}

	fmt.Printf("Transaction sent successfully! Transaction hash: %s\n", signedTx.Hash().Hex())
	fmt.Println("Please check the transaction status on the blockchain explorer")

}
