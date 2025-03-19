package main

import (
	"fmt"
	"log"

	"github.com/mer-coder/meson-bridge/pkg/meson"
)

func main() {
	// 创建客户端
	client := meson.NewClient()

	// 构建请求
	req := &meson.PriceRequest{
		From:        "merlin:67", // Merlin链上的MBTC
		To:          "zksync:67", // ZKsync链上的MBTC
		Amount:      "0.0001",
		FromAddress: "0x742d35Cc6634C0532925a3b844Bc454e4438f44e", // 任意示例地址
	}

	// 获取价格
	resp, err := client.GetPrice(req)
	if err != nil {
		log.Fatalf("获取价格失败: %v", err)
	}

	// 打印结果
	fmt.Println("跨链价格信息:")
	fmt.Printf("- 费用: %s\n", resp.Fee)
	fmt.Printf("- 预计时间: %d秒\n", resp.EstimatedTime)
	fmt.Printf("- 最小金额: %s\n", resp.MinAmount)
	fmt.Printf("- 最大金额: %s\n", resp.MaxAmount)
}
