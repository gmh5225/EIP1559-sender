package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math"
	"math/big"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
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
	tokenContractFlag := flag.String("tokenContract", "", "Token contract address (optional, if not provided, ETH will be transferred)")
	tokenABIFlag := flag.String("tokenABI", "", "Token ABI JSON string (required only for ERC20 transfers)")

	// Add usage information
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [options]\n", os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(), "\nOptions:\n")
		flag.PrintDefaults()
		fmt.Fprintf(flag.CommandLine.Output(), "\nExample for ETH transfer:\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  %s -privateKey 0x... -receiver 0x... -rpcURL https://... -chainID 1 -tokenValue 0.1\n", os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(), "\nExample for ERC20 transfer:\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  %s -privateKey 0x... -receiver 0x... -rpcURL https://... -chainID 1 -tokenValue 0.1 -tokenContract 0x... -tokenABI '[...]'\n", os.Args[0])
	}

	flag.Parse()

	// Check if required parameters are provided
	if *privateKeyFlag == "" || *receiverFlag == "" || *rpcURLFlag == "" || *tokenValueFlag == 0 {
		fmt.Println("Error: Missing required parameters")
		flag.Usage()
		os.Exit(1)
	}

	// Check if tokenContract and tokenABI are provided together for ERC20 transfers
	if (*tokenContractFlag == "" && *tokenABIFlag != "") || (*tokenContractFlag != "" && *tokenABIFlag == "") {
		fmt.Println("Error: Both tokenContract and tokenABI must be provided for ERC20 transfers")
		flag.Usage()
		os.Exit(1)
	}

	// Connect to RPC URL
	client, err := ethclient.Dial(*rpcURLFlag)
	if err != nil {
		log.Fatalf("Failed to connect to the RPC URL: %v", err)
	}
	fmt.Printf("Connected to the RPC URL %s\n", *rpcURLFlag)

	// Get chain ID
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

	privateKey, err := crypto.HexToECDSA(*privateKeyFlag)
	if err != nil {
		log.Fatalf("Failed to parse private key: %v", err)
	}

	fromAddress := crypto.PubkeyToAddress(privateKey.PublicKey)
	toAddress := common.HexToAddress(*receiverFlag)
	fmt.Printf("Sender's address: %s\n", fromAddress.Hex())
	fmt.Printf("Receiver address: %s\n", toAddress.Hex())

	nonce, err := client.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		log.Fatalf("Failed to get nonce: %v", err)
	}
	fmt.Println("nonce:", nonce)

	var data []byte
	var tokenAddress common.Address
	var transferAmount *big.Int

	if *tokenContractFlag == "" {
		// ETH transfer
		tokenValue := *tokenValueFlag
		weiPerToken := big.NewInt(1e18)
		tokenValueBigFloat := new(big.Float).SetFloat64(tokenValue)
		transferAmount, _ := new(big.Float).Mul(tokenValueBigFloat, new(big.Float).SetInt(weiPerToken)).Int(nil)
		fmt.Printf("Transfer amount: %.6f tokens (equivalent to %s Wei)\n", tokenValue, transferAmount.String())

	} else {
		// ERC20 transfer
		tokenAddress = common.HexToAddress(*tokenContractFlag)
		parsedABI, err := abi.JSON(strings.NewReader(*tokenABIFlag))
		if err != nil {
			log.Fatalf("Failed to parse ABI: %v", err)
		}

		decimals, err := getTokenDecimals(client, tokenAddress)
		if err != nil {
			log.Fatalf("Failed to get token decimals: %v", err)
		}

		tokenAmount := new(big.Float).Mul(big.NewFloat(*tokenValueFlag), new(big.Float).SetFloat64(math.Pow10(int(decimals))))
		transferAmount, _ = tokenAmount.Int(nil)
		fmt.Printf("Transferring ERC20 token: %f (base units: %s)\n", *tokenValueFlag, transferAmount.String())

		data, err = parsedABI.Pack("transfer", toAddress, transferAmount)
		if err != nil {
			log.Fatalf("Failed to pack transfer data: %v", err)
		}
	}

	// Get gas price information
	gasTipCap, err := client.SuggestGasTipCap(context.Background())
	if err != nil {
		log.Fatalf("Failed to suggest gas tip cap: %v", err)
	}

	header, err := client.HeaderByNumber(context.Background(), nil)
	if err != nil {
		log.Fatalf("Failed to get latest header: %v", err)
	}

	gasFeeCap := new(big.Int).Add(
		header.BaseFee,
		new(big.Int).Mul(gasTipCap, big.NewInt(2)),
	)

	// Estimate gas limit
	var gasLimit uint64
	if *tokenContractFlag == "" {
		gasLimit = 21000 // Standard gas limit for ETH transfers
	} else {
		gasLimit, err = client.EstimateGas(context.Background(), ethereum.CallMsg{
			From: fromAddress,
			To:   &tokenAddress,
			Data: data,
		})
		if err != nil {
			log.Fatalf("Failed to estimate gas: %v", err)
		}
	}
	fmt.Printf("Gas limit: %d\n", gasLimit)

	// Create EIP-1559 transaction
	var tx *types.Transaction
	if *tokenContractFlag == "" {
		// ETH transfer
		tx = types.NewTx(&types.DynamicFeeTx{
			ChainID:   chainID,
			Nonce:     nonce,
			GasTipCap: gasTipCap,
			GasFeeCap: gasFeeCap,
			Gas:       gasLimit,
			To:        &toAddress,
			Value:     transferAmount,
			Data:      nil,
		})
	} else {
		// ERC20 transfer
		tx = types.NewTx(&types.DynamicFeeTx{
			ChainID:   chainID,
			Nonce:     nonce,
			GasTipCap: gasTipCap,
			GasFeeCap: gasFeeCap,
			Gas:       gasLimit,
			To:        &tokenAddress,
			Value:     big.NewInt(0),
			Data:      data,
		})
	}

	// Sign transaction
	signedTx, err := types.SignTx(tx, types.LatestSignerForChainID(chainID), privateKey)
	if err != nil {
		log.Fatalf("Failed to sign transaction: %v", err)
	}

	// Send transaction
	err = client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		log.Fatalf("Failed to send transaction: %v", err)
	}

	fmt.Printf("Transaction sent successfully! Transaction hash: %s\n", signedTx.Hash().Hex())
	fmt.Println("Please check the transaction status on the blockchain explorer")
}

func getTokenDecimals(client *ethclient.Client, tokenAddress common.Address) (uint8, error) {
	data := []byte{0x31, 0x3c, 0xe5, 0x67} // Correct byte order for "decimals()" function selector
	msg := ethereum.CallMsg{To: &tokenAddress, Data: data}
	result, err := client.CallContract(context.Background(), msg, nil)
	if err != nil {
		return 0, err
	}
	return result[len(result)-1], nil // The result is a uint8, padded to 32 bytes, so we take the last byte
}
