package main

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"log"
	"math/big"
	"os"

	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

/*
*
任务 2：合约代码生成 任务目标
使用 abigen 工具自动生成 Go 绑定代码，用于与 Sepolia 测试网络上的智能合约进行交互。 具体任务

编写智能合约
使用 Solidity 编写一个简单的智能合约，例如一个计数器合约。
编译智能合约，生成 ABI 和字节码文件。
使用 abigen 生成 Go 绑定代码
安装 abigen 工具。
使用 abigen 工具根据 ABI 和字节码文件生成 Go 绑定代码。
使用生成的 Go 绑定代码与合约交互
编写 Go 代码，使用生成的 Go 绑定代码连接到 Sepolia 测试网络上的智能合约。
调用合约的方法，例如增加计数器的值。
输出调用结果。
*/
func main() {
	contractAddress := common.HexToAddress("0xbb5bBD34A25e700111628E5b0bE9d6Ddd2D425F4")
	rpcURL := os.Getenv("ETH_RPC_URL")
	rpcURL = "https://ethereum-sepolia-rpc.publicnode.com"
	ctx := context.Background()
	client, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		log.Fatalf("failed to get client : %v", err)
	}
	instance, err := NewCounter(contractAddress, client)
	if err != nil {
		log.Fatalf("failed to get NewCounter : %v", err)
	}
	fmt.Println("正在读取当前计数器...")
	count, err := instance.GetCount(&bind.CallOpts{Context: ctx})
	if err != nil {
		log.Fatalf("failed to get GetCount : %v", err)
	}
	fmt.Printf("当前计数器值为: %v\n", count)
	fmt.Printf("===================================================")
	// ================= 写操作（Send Transaction） =================
	fmt.Println("正在增加计数器...")
	privKeyHex := "812596831af721d5daf6f637e3abd89cade58eea1c7cb201909f92deabb85633"
	privateKey, err := crypto.HexToECDSA(privKeyHex)
	if err != nil {
		log.Fatalf("failed to get HexToECDSA : %v", err)
	}
	publicKey := privateKey.Public()
	publicKeyECDSA, _ := publicKey.(*ecdsa.PublicKey)
	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)

	// 获取链 ID
	chainID, _ := client.ChainID(ctx)
	// 获取 Nonce
	nonce, _ := client.PendingNonceAt(ctx, fromAddress)
	// 构造授权对象
	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
	if err != nil {
		log.Fatalf("failed to get NewKeyedTransactorWithChainID : %v", err)
	}
	auth.Nonce = big.NewInt(int64(nonce))
	auth.GasLimit = uint64(300000)
	// 发送交易
	tx, err := instance.Increment(auth)
	if err != nil {
		log.Fatalf("failed to increment : %v", err)
	}
	fmt.Printf("交易已发送，哈希: %s\n", tx.Hash().Hex())

	count, err = instance.GetCount(&bind.CallOpts{Context: ctx})
	if err != nil {
		log.Fatalf("line 87 failed to get GetCount : %v", err)
	}
	fmt.Printf("当前计数器值为: %v\n", count)
}
