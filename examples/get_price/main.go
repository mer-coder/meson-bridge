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
		To:          "duck:67", // ZKsync链上的MBTC
		Amount:      "0.01",
	}

	// 获取价格
	resp, err := client.GetPrice(req)
	if err != nil {
		log.Fatalf("获取价格失败: %v", err)
	}

	// 打印结果
	fmt.Println("跨链价格信息:", resp)
}
