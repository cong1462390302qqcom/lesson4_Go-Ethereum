package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum/go-ethereum/ethclient"
)

// 01-subscribe-blocks.go
// 通过 SubscribeNewHead 订阅新区块头。
// 注意：大多数节点要求使用 WebSocket RPC，例如：ws://127.0.0.1:8546 或 wss://...
func main() {
	ETH_WS_URL := "wss://eth-sepolia.g.alchemy.com/v2/alch_sTh0kkpSfvW4BZ0jojhpO"
	//ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	client, err := ethclient.DialContext(ctx, ETH_WS_URL)
	if err != nil {
		log.Fatalf("failed to connect to Ethereum node: %v", err)
	}
	defer client.Close()
	//make建立通道，make:初始化并创建 Go 语言中特有的三种引用类型——切片（Slice）、映射（Map）和通道（Channel）
	headers := make(chan *types.Header)
	//发送订阅信号给节点,每次区块创建新的时会通知
	sub, err := client.SubscribeNewHead(ctx, headers)
	if err != nil {
		log.Fatalf("failed to subscribe new heads: %v", err)
	}
	fmt.Printf("Subscribed to new blocks via %s\n", ETH_WS_URL)
	defer sub.Unsubscribe()
	// 捕获 Ctrl+C 退出
	sigCh := make(chan os.Signal, 1)                      //创建接受系统信号
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM) //Signal Interrupt-中断信号 Signal Terminate-终止信号

	for {
		select {
		case header := <-headers:
			// 只要节点推来新块，就会执行这里
			if header == nil {
				continue
			}
			fmt.Printf("[%s] New Block - Number: %d, Hash: %s\n", time.Now().Format(time.RFC3339), header.Number.Uint64(), header.Hash().Hex())
		case err := <-sub.Err():
			log.Printf("subscription error: %v", err)
			return
		case sig := <-sigCh:
			fmt.Printf("received signal %s, shutting down...\n", sig.String())
			return
		case <-ctx.Done(): //外部超时取消
			fmt.Println("Done!")
			return
		}
	}
}
