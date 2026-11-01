package main

import (
	"context"
	"crypto/ecdsa"
	"flag"
	"fmt"
	"log"
	"math/big"
	"os"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

/*
*
任务 1：区块链读写 任务目标
使用 Sepolia 测试网络实现基础的区块链交互，包括查询区块和发送交易。 具体任务

环境搭建
安装必要的开发工具，如 Go 语言环境、 go-ethereum 库。
注册 Infura 账户，获取 Sepolia 测试网络的 API Key。

1 查询区块
编写 Go 代码，使用 ethclient 连接到 Sepolia 测试网络。
实现查询指定区块号的区块信息，包括区块的哈希、时间戳、交易数量等。
输出查询结果到控制台。
2 发送交易
准备一个 Sepolia 测试网络的以太坊账户，并获取其私钥。
编写 Go 代码，使用 ethclient 连接到 Sepolia 测试网络。
构造一笔简单的以太币转账交易，指定发送方、接收方和转账金额。
对交易进行签名，并将签名后的交易发送到网络。
输出交易的哈希值。
*/
func main() {
	blockNumberFlag := flag.Uint64("number", 0, "block number to query (0 means skip)")
	toAddrHex := flag.String("to", "", "recipient address (required for send mode)")
	amountEth := flag.Float64("amount", 0, "amount in ETH (required for send mode)")
	flag.Parse()
	if *toAddrHex == "" {
		fmt.Println("to conectTest()")
		conectTest(*blockNumberFlag)
	} else {
		fmt.Println("to sendTransactionTest()")
		sendTransactionTest(*toAddrHex, *amountEth)
	}
}

func conectTest(blockNumberFlag uint64) {
	rpcURL := os.Getenv("ETH_RPC_URL")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	client, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		log.Fatalf("fail to connect to Ethereum node: %v", err)
	}
	var block *types.Block
	if blockNumberFlag > 0 {
		//查询指定区块
		block, err = client.BlockByNumber(ctx, new(big.Int).SetUint64(blockNumberFlag))
		if err != nil {
			log.Fatalf("failed to get %d block: %v", blockNumberFlag, err)
		}
	} else {
		//查询最新区块
		block, err = client.BlockByNumber(ctx, nil)
		if err != nil {
			log.Fatalf("failed to get %d block: %v", blockNumberFlag, err)
		}
	}
	// 基本信息
	fmt.Printf("Number       : %d\n", block.Number().Uint64())
	fmt.Printf("Hash         : %s\n", block.Hash().Hex())
	fmt.Printf("Parent Hash  : %s\n", block.ParentHash().Hex())
	// 时间信息
	blockTime := time.Unix(int64(block.Time()), 0)
	fmt.Printf("Time         : %s\n", blockTime.Format(time.RFC3339))
	fmt.Printf("Time (Local) : %s\n", blockTime.Local().Format("2006-01-02 15:04:05 MST"))
	//交易数量
	txCount := len(block.Transactions())
	fmt.Printf("Tx Count     : %d\n", txCount)
}

func sendTransactionTest(toAddrHex string, amountEth float64) {
	rpcURL := os.Getenv("ETH_RPC_URL")
	privKeyHex := os.Getenv("ETH_SENDER_PRIVATE_KEY")

	fmt.Println(privKeyHex)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	client, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		log.Fatalf("fail to connect to Ethereum node: %v", err)
	}
	//解析私钥，获取发送方地址
	privateKey, err := crypto.HexToECDSA(privKeyHex)
	if err != nil {
		log.Fatalf("invalid private key: %v", err)
	}
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		log.Fatal("error casting public key to ECDSA")
	}

	//发送交易的必须参数(7个) chainid,nonce,to,value,gasLimit,gasPrice,data

	fromAddr := crypto.PubkeyToAddress(*publicKeyECDSA)
	toAddr := common.HexToAddress(toAddrHex)
	chainId, err := client.ChainID(ctx)
	if toAddr == common.HexToAddress("0x0000000000000000000000000000000000000000") {
		log.Fatalf("error toAddr : %v", toAddr)
		return
	}
	if err != nil {
		log.Fatalf("failed to get chain id: %v", err)
	}
	nonce, err := client.PendingNonceAt(ctx, fromAddr)
	if err != nil {
		log.Fatalf("failed to get nonce: %v", err)
	}
	amountWei := new(big.Float).Mul(big.NewFloat(amountEth), big.NewFloat(1e18))
	//金额去小数位
	valueWei, _ := amountWei.Int(nil)
	gasTipCap, err := client.SuggestGasTipCap(ctx)
	if err != nil {
		log.Fatalf("failed to get GasTipCap: %v", err)
	}
	gasFeeCap, err := client.SuggestGasPrice(ctx)
	if err != nil {
		log.Fatalf("failed to get GasPrice: %v", err)
	}
	msg := ethereum.CallMsg{From: fromAddr,
		To:    &toAddr,       // 如果调合约，这里是合约地址
		Data:  nil,           // 如果调合约，这里是 ABI 编码后的数据
		Value: big.NewInt(0), // 如果带 ETH 就填金额
	}
	estimateGasLimit, err := client.EstimateGas(ctx, msg)
	if err != nil {
		log.Fatalf("failed to EstimateGas: %v", err)
	}
	gasLimit := estimateGasLimit * 120 / 100
	txData := &types.DynamicFeeTx{
		ChainID:   chainId,
		Nonce:     nonce,
		To:        &toAddr,
		Value:     valueWei,
		GasTipCap: gasTipCap,
		GasFeeCap: gasFeeCap,
		Gas:       gasLimit,
		Data:      nil,
	}
	//交易内容加签
	tx := types.NewTx(txData)
	signer := types.NewLondonSigner(chainId)
	signedTx, err := types.SignTx(tx, signer, privateKey)
	if err != nil {
		log.Fatalf("failed to sign transaction: %v", err)
	}
	if err := client.SendTransaction(ctx, signedTx); err != nil {
		log.Fatalf("failed to send transaction: %v", err)
	}
	// 输出交易信息
	fmt.Println("=== Transaction Sent ===")
	fmt.Printf("From       : %s\n", fromAddr.Hex())
	fmt.Printf("To         : %s\n", toAddr.Hex())
	fmt.Printf("Value      : %s ETH (%s Wei)\n", fmt.Sprintf("%.6f", amountEth), valueWei.String())
	fmt.Printf("Gas Limit  : %d\n", gasLimit)
	fmt.Printf("estimateGasLimit : %d\n", estimateGasLimit)
	fmt.Printf("Gas Tip Cap: %s Wei\n", gasTipCap.String())
	fmt.Printf("Gas Fee Cap: %s Wei\n", gasFeeCap.String())
	fmt.Printf("Nonce      : %d\n", nonce)
	fmt.Printf("Tx Hash    : %s\n", signedTx.Hash().Hex())
	fmt.Println("\nTransaction is pending. Use --tx flag to query status:")
	fmt.Printf("  go run main.go --tx %s\n", signedTx.Hash().Hex())

}
