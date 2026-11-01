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

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

// 03-tx-ops.go
// 支持两种操作模式：
// 1. 查询交易：--tx <hash> - 按哈希查询交易与回执，解析关键字段
// 2. 发送交易：--send --to <address> --amount <eth> - 发起 ETH 转账交易

// go run .\main.go --tx 0x4a70a384867b44a3cd8af924e03eabd1c2d34ccab246aaed98b5747b1bac9bf2  --to 0x0041d889909865a507502817bD43063A826E230d
// 0x4a70a384867b44a3cd8af924e03eabd1c2d34ccab246aaed98b5747b1bac9bf2
//
// go run .\main.go -send -to 0xD19147073e0d428ebB10A5Da2754e1DD846bdaC1  -amount 0.001
// 0x0ce50ab36bcfc25bbffa2bc6ca5b819e5067c61b40927187a8a3d5ba796bf91b
func main() {
	txHashHex := flag.String("tx", "", "transaction hash (for query mode)")
	sendMode := flag.Bool("send", false, "enable send transaction mode")
	toAddrHex := flag.String("to", "", "recipient address (required for send mode)")
	amountEth := flag.Float64("amount", 0, "amount in ETH (required for send mode)")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 判断操作模式
	if *sendMode {
		// 发送交易模式
		if *toAddrHex == "" || *amountEth <= 0 {
			log.Fatal("send mode requires --to and --amount flags")
		}
		sendTransaction(ctx, *toAddrHex, *amountEth)
	} else {
		// 查询交易模式
		if *txHashHex == "" {
			log.Fatal("query mode requires --tx flag, or use --send for send mode")
		}
		queryTransaction(*txHashHex)
	}

}

// 查询交易
func queryTransaction(toAddrHex string) {
	client := connectRPCURL()
	defer client.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	txHash := common.HexToHash(toAddrHex)
	tx, isPending, err := client.TransactionByHash(ctx, txHash)
	if err != nil {
		log.Fatalf("failed to get transaction: %v", err)
	}
	fmt.Println("=== Transaction ===")
	printTxBasicInfo(tx, isPending)

	receipt, err := client.TransactionReceipt(ctx, txHash)
	if err != nil {
		log.Printf("failed to get receipt (maybe pending): %v", err)
		return
	}
	fmt.Println("=== Receipt ===")
	printReceiptInfo(receipt)
}

// 发送交易
func sendTransaction(ctx context.Context, toAddrHex string, amountEth float64) {
	client := connectRPCURL()
	defer client.Close()
	privKeyHex := os.Getenv("ETH_SENDER_PRIVATE_KEY")
	if privKeyHex == "" {
		log.Fatal("SENDER_PRIVATE_KEY is not set (required for send mode)")
	}
	//解析私钥，获取发送方地址
	privKey, err := crypto.HexToECDSA(trim0x(privKeyHex))
	if err != nil {
		log.Fatalf("invalid private key: %v", err)
	}

	//publicKey :=privKey.PublicKey //也能用，但是后期密码学可能扩展，类型断言最好提前使用
	publicKey := privKey.Public()
	//类型断言
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		log.Fatal("error casting public key to ECDSA")
	}
	fromAddr := crypto.PubkeyToAddress(*publicKeyECDSA)
	toAddr := common.HexToAddress(toAddrHex)

	// 获取链 ID
	chainId, err := client.ChainID(ctx)
	if err != nil {
		log.Fatalf("failed to get chain id: %v", err)
	}
	// 获取 nonce
	nonce, err := client.PendingNonceAt(ctx, fromAddr)
	if err != nil {
		log.Fatalf("failed to get nonce: %v", err)
	}
	// 获取建议的 Gas 价格（使用 EIP-1559 动态费用）
	gasTipCap, err := client.SuggestGasTipCap(ctx) //它只返回小费（Priority Fee / Tip）的建议值
	if err != nil {
		log.Fatalf("failed to get gas tip cap: %v", err)
	}
	// 获取 base fee，计算 fee cap
	header, err := client.HeaderByNumber(ctx, nil)
	baseFee := header.BaseFee
	if baseFee == nil {
		// 如果不支持 EIP-1559，使用传统 gasprice
		gasPrice, err := client.SuggestGasPrice(ctx) //它返回的是一个总价（Gas Price）“基础费 + 小费”
		if err != nil {
			log.Fatalf("failed to get gas price: %v", err)
		}
		baseFee = gasPrice
	}
	// fee cap = base fee * 2 + tip cap（简单策略）
	gasFeeCap := new(big.Int).Add(new(big.Int).Mul(baseFee, big.NewInt(2)), gasTipCap)

	// 估算 Gas Limit（普通转账固定为 21000）
	gasLimit := uint64(21000)

	// 转换 ETH 金额为 Wei
	// amountEth * 1e18
	amountWei := new(big.Float).Mul(big.NewFloat(amountEth), big.NewFloat(1e18))
	valueWei, _ := amountWei.Int(nil)

	// 检查余额是否足够
	balance, err := client.BalanceAt(ctx, fromAddr, nil)
	if err != nil {
		log.Fatalf("failed to get balance: %v", err)
	}
	// 计算总费用：value + gasFeeCap * gasLimit
	totalCost := new(big.Int).Add(valueWei, new(big.Int).Mul(gasFeeCap, big.NewInt(int64(gasLimit))))
	if balance.Cmp(totalCost) < 0 {
		log.Fatalf("insufficient balance: have %s wei, need %s wei", balance.String(), totalCost.String())
	}
	// 构造交易（EIP-1559 动态费用交易）
	txData := &types.DynamicFeeTx{
		ChainID:   chainId,
		Nonce:     nonce,
		GasTipCap: gasTipCap,
		GasFeeCap: gasFeeCap,
		Gas:       gasLimit,
		To:        &toAddr,
		Value:     valueWei,
		Data:      nil,
	}
	tx := types.NewTx(txData)
	//签名
	singer := types.NewLondonSigner(chainId)
	signedTx, err := types.SignTx(tx, singer, privKey)
	if err != nil {
		log.Fatalf("failed to sign transaction: %v", err)
	}
	//发送
	err = client.SendTransaction(ctx, signedTx)
	if err != nil {
		log.Fatalf("failed to send transaction: %v", err)
	}
	// 输出交易信息
	fmt.Println("=== Transaction Sent ===")
	fmt.Printf("From       : %s\n", fromAddr.Hex())
	fmt.Printf("To         : %s\n", toAddr.Hex())
	fmt.Printf("Value      : %s ETH (%s Wei)\n", fmt.Sprintf("%.6f", amountEth), valueWei.String())
	fmt.Printf("Gas Limit  : %d\n", gasLimit)
	fmt.Printf("Gas Tip Cap: %s Wei\n", gasTipCap.String())
	fmt.Printf("Gas Fee Cap: %s Wei\n", gasFeeCap.String())
	fmt.Printf("Nonce      : %d\n", nonce)
	fmt.Printf("Tx Hash    : %s\n", signedTx.Hash().Hex())
	fmt.Println("\nTransaction is pending. Use --tx flag to query status:")
	fmt.Printf("  go run main.go --tx %s\n", signedTx.Hash().Hex())
}

func connectRPCURL() *ethclient.Client {
	rpcURL := os.Getenv("ETH_RPC_URL")
	if rpcURL == "" {
		log.Fatal("ETH_RPC_URL is not set")
	}
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		log.Fatalf("failed to connect to Ethereum node: %v", err)
	}
	return client
}

func trim0x(s string) string {
	if len(s) > 2 && s[:2] == "0x" {
		return s[2:]
	}
	return s
}

func printTxBasicInfo(tx *types.Transaction, isPending bool) {
	fmt.Printf("Hash        : %s\n", tx.Hash().Hex())
	fmt.Printf("Nonce       : %d\n", tx.Nonce())
	fmt.Printf("Gas         : %d\n", tx.Gas())
	fmt.Printf("Gas Price   : %s\n", tx.GasPrice().String())
	fmt.Printf("To          : %v\n", tx.To())
	fmt.Printf("Value (Wei) : %s\n", tx.Value().String())
	fmt.Printf("Data Len    : %d bytes\n", len(tx.Data()))
	fmt.Printf("Pending     : %v\n", isPending)
}

func printReceiptInfo(r *types.Receipt) {
	fmt.Printf("Status      : %d\n", r.Status)
	fmt.Printf("BlockNumber : %d\n", r.BlockNumber.Uint64())
	fmt.Printf("BlockHash   : %s\n", r.BlockHash.Hex())
	fmt.Printf("TxIndex     : %d\n", r.TransactionIndex)
	fmt.Printf("Gas Used    : %d\n", r.GasUsed)
	fmt.Printf("Logs        : %d\n", len(r.Logs))
	if len(r.Logs) > 0 {
		fmt.Printf("First Log Address : %s\n", r.Logs[0].Address.Hex())
	}
}
