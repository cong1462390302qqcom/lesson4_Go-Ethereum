package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math"
	"math/big"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

//0xD19147073e0d428ebB10A5Da2754e1DD846bdaC1
// 04-account-balance.go
// 查询账户 ETH 余额（Wei 与 ETH）。
func main() {
	addrHex := flag.String("address", "", "account address (required)")
	blockNumber := flag.Int64("block", -1, "block number to query (-1 means latest)")
	flag.Parse()

	if *addrHex == "" {
		log.Fatal("missing --address flag")
	}
	rpcURL := os.Getenv("ETH_RPC_URL")
	if rpcURL == "" {
		log.Fatal("ETH_RPC_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	client, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		log.Fatalf("failed to connect to Ethereum node: %v", err)
	}
	defer client.Close()
	address := common.HexToAddress(*addrHex)
	var blockNum *big.Int
	if *blockNumber > 0 {
		blockNum = big.NewInt(*blockNumber)
	}
	balanceWei, err := client.BalanceAt(ctx, address, blockNum)
	if err != nil {
		log.Fatalf("failed to get balance: %v", err)
	}
	fmt.Println("=== Account Balance ===")
	fmt.Printf("Address     : %s\n", address.Hex())
	if blockNum == nil {
		fmt.Printf("Block       : latest\n")
	} else {
		fmt.Printf("Block       : %d\n", blockNum.Uint64())
	}
	fmt.Printf("Balance Wei : %s\n", balanceWei.String())

	balanceFloat := new(big.Float).SetInt(balanceWei)
	ethValue := new(big.Float).Quo(balanceFloat, big.NewFloat(math.Pow10(18)))
	fmt.Printf("Balance ETH : %f\n", ethValue)

}
