package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"log"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
)

func main() {
	//ETH_RPC_URL="http://127.0.0.1:8545"
	//ETH_RPC_URL="https://mainnet.infura.io/v3/5af834d6600f480c8105011d86525449"
	//https://sepolia.infura.io/v3/alch_TOqMhTJ36AdZHJOyP8TpZ
	//rpcURL := os.Getenv("ETH_RPC_URL")
	rpcURL := "https://eth-sepolia.g.alchemy.com/v2/alch_TOqMhTJ36AdZHJOyP8TpZ"
	if rpcURL == "" {
		log.Fatal("Please provide a valid RPC URL")
	}
	//整个交易的超时时间
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	//使用dialcontext连接节点
	client, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		log.Fatalf("failed to connect to Ethereum node: %v", err)
	}
	defer client.Close()
	chainID, err := client.ChainID(ctx)
	if err != nil {
		log.Fatalf("failed to get chain id: %v", err)
	}
	header, err := client.HeaderByNumber(ctx, nil)
	if err != nil {
		log.Fatalf("failed to get latest block header: %v", err)
	}
	fmt.Println("=== Ethereum Node Info ===")
	fmt.Printf("RPC URL       : %s\n", rpcURL)
	fmt.Printf("Chain ID      : %s\n", chainID.String())
	fmt.Println("\n⚠️  注意: 'Latest' 区块是节点当前认为的最新区块，可能尚未被所有节点确认")
	fmt.Println("   不同RPC节点可能返回不同的 'latest' 区块，导致与浏览器不匹配")
	fmt.Println("   建议对比 'Safe' 或 'Finalized' 区块（已确认的区块）")
	fmt.Println()
	fmt.Printf("Latest Block  : %d\n", header.Number.Uint64())
	fmt.Printf("Block Hash    : %s\n", header.Hash().Hex())
	fmt.Printf("Block Time    : %s\n", time.Unix(int64(header.Time), 0).Format(time.RFC3339))
	fmt.Println("==========================")

	// 示例：也可以获取任意指定高度的区块头
	if header.Number.Uint64() > 0 {
		//获取当前区块的父区块
		num := new(big.Int).Sub(header.Number, big.NewInt(1))
		prevHeader, err := client.BlockByNumber(ctx, num)
		if err == nil {
			fmt.Printf("Prev Block    : %d (%s)\n", prevHeader.Number().Uint64(), prevHeader.Hash().Hex())
		}
	}
	// 查询 safe 区块（浏览器通常显示这个）
	safeHeader, safeHash, err := getBlockByTag(ctx, client, "safe")
	if err != nil {
		log.Printf("failed to get safe block: %v (this may not be supported by all nodes)", err)
	} else {
		fmt.Println("\n=== Safe Block (推荐对比) ===")
		fmt.Printf("Block Number  : %d\n", safeHeader.Number.Uint64())
		fmt.Printf("Block Hash    : %s (RPC提供的hash，与浏览器一致)\n", safeHash.Hex())
		fmt.Printf("Calculated    : %s (计算出的hash，可能不匹配)\n", safeHeader.Hash().Hex())
		fmt.Printf("Block Time    : %s\n", time.Unix(int64(safeHeader.Time), 0).Format(time.RFC3339))
		fmt.Printf("Confirmations : %d\n", header.Number.Uint64()-safeHeader.Number.Uint64())
		fmt.Println("=============================")
	}
	finalizedHeader, finalizedHash, err := getBlockByTag(ctx, client, "finalized")
	if err != nil {
		log.Printf("failed to get finalized block: %v (this may not be supported by all nodes)", err)
	} else {
		fmt.Println("\n=== Finalized Block  ===")
		fmt.Printf("Block Number  : %d\n", finalizedHeader.Number.Uint64())
		fmt.Printf("Block Hash    : %s (RPC提供的hash，与浏览器一致)\n", finalizedHash.Hex())
		fmt.Printf("Calculated    : %s (计算出的hash，可能不匹配)\n", finalizedHeader.Hash().Hex())
		fmt.Printf("Block Time    : %s\n", time.Unix(int64(finalizedHeader.Time), 0).Format(time.RFC3339))
		fmt.Printf("Confirmations : %d\n", header.Number.Uint64()-finalizedHeader.Number.Uint64())
		fmt.Println("=============================")
	}

}

// getBlockByTag 查询指定标签的区块头（safe, finalized, latest 等）
// 返回 Header、RPC 提供的 Hash 和错误
// 注意：需要使用底层 RPC 调用，因为 ethclient 的高级 API 不直接支持使用（safe, finalized, latest 等）这些标签
func getBlockByTag(ctx context.Context, client *ethclient.Client, tag string) (*types.Header, common.Hash, error) {
	rpcClient := client.Client() //获取底层rpc客户端
	var raw json.RawMessage
	//使用底层调用发送原生方法请求
	err := rpcClient.CallContext(ctx, &raw, "eth_getBlockByNumber", tag, false)
	if err != nil {
		return nil, common.Hash{}, fmt.Errorf("RPC call failed: %w", err)
	}
	if len(raw) == 0 || string(raw) == "" {
		return nil, common.Hash{}, fmt.Errorf("%s block not found", tag)
	}

	var blockData struct {
		Number      string         `json:"number"`
		Hash        common.Hash    `json:"hash"`
		ParentHash  common.Hash    `json:"parentHash"`
		UncleHash   common.Hash    `json:"sha3Uncles"`
		Coinbase    common.Address `json:"miner"`
		Root        common.Hash    `json:"stateRoot"`
		TxHash      common.Hash    `json:"transactionsRoot"`
		ReceiptHash common.Hash    `json:"receiptsRoot"`
		Bloom       hexutil.Bytes  `json:"logsBloom"`
		Difficulty  *hexutil.Big   `json:"difficulty"`
		GasLimit    hexutil.Uint64 `json:"gasLimit"`
		GasUsed     hexutil.Uint64 `json:"gasUsed"`
		Time        hexutil.Uint64 `json:"timestamp"`
		Extra       hexutil.Bytes  `json:"extraData"`
		MixDigest   common.Hash    `json:"mixHash"`
		Nonce       hexutil.Bytes  `json:"nonce"`
		BaseFee     *hexutil.Big   `json:"baseFeePerGas"`
	}

	//解析区块内容
	if err := json.Unmarshal(raw, &blockData); err != nil {
		return nil, common.Hash{}, fmt.Errorf("JSON unmarshal failed: %w", err)
	}
	//0x十六进制数 转换成十进制
	num, ok := new(big.Int).SetString(blockData.Number[2:], 16)
	if !ok {
		return nil, common.Hash{}, fmt.Errorf("invalid block number: %s", blockData.Number)
	} else {
		fmt.Printf("getBlockByTag Block Number  : %d\n", blockData.Number)
	}

	header := &types.Header{
		ParentHash:  blockData.ParentHash,
		UncleHash:   blockData.UncleHash,
		Coinbase:    blockData.Coinbase,
		Root:        blockData.Root,
		TxHash:      blockData.TxHash,
		ReceiptHash: blockData.ReceiptHash,
		Bloom:       types.Bloom(blockData.Bloom),
		Difficulty:  (*big.Int)(blockData.Difficulty),
		Number:      num,
		GasLimit:    uint64(blockData.GasLimit),
		GasUsed:     uint64(blockData.GasUsed),
		Time:        uint64(blockData.Time),
		Extra:       blockData.Extra,
		MixDigest:   blockData.MixDigest,
		Nonce:       types.BlockNonce(blockData.Nonce),
		BaseFee:     (*big.Int)(blockData.BaseFee),
	}
	return header, blockData.Hash, nil
}
