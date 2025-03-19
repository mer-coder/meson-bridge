# Meson 跨链桥 Go SDK

这是一个用于与 Meson 跨链桥交互的 Go 语言 SDK，可以帮助开发者轻松实现跨链转账功能。

## 功能特性

- 获取跨链转账费用和最大/最小金额限制
- 编码和签名跨链交易
- 提交跨链交易
- 查询跨链交易状态
- 生成必要的 ERC20 代币授权交易

## 安装

```bash
go get github.com/mer-coder/meson-bridge
```

## 使用示例

```go
package main

import (
    "context"
    "fmt"
    "github.com/shopspring/decimal"
    "github.com/mer-coder/meson-bridge/pkg/meson"
)

func main() {
    // 创建一个新的Bridge实例
    bridge := meson.NewBridge()
    
    // 获取跨链费用
    price, err := bridge.client.GetPrice(&meson.PriceRequest{
        From:        "merlin:67", // 从Merlin链上的MBTC
        To:          "zksync:67", // 到zksync链上的MBTC
        Amount:      "0.0001",    // 金额
        FromAddress: "0xYourFromAddress", 
    })
    if err != nil {
        fmt.Printf("获取费用失败: %v\n", err)
        return
    }
    
    fmt.Printf("跨链费用: %s\n", price.Fee)
    fmt.Printf("预计时间: %d秒\n", price.EstimatedTime)
}
```

## 完整跨链流程

1. 授权跨链桥合约使用你的代币
   ```go
   approveTxData, _ := bridge.GetApproveData()
   // 发送授权交易
   ```

2. 准备跨链交易
   ```go
   resp, _ := bridge.BridgeMBTC(
       context.Background(),
       decimal.NewFromFloat(0.0001),
       "0xFromAddress",
       "0xToAddress",
       meson.ChainMerlin,
       meson.ChainZksync,
       meson.TokenMBTC,
       meson.TokenMBTC,
   )
   ```

3. 签名消息
   ```go
   signature, _ := signData(privateKey, common.FromHex(resp.SigningRequest.Hash))
   ```

4. 提交跨链交易
   ```go
   swapId, _ := bridge.SubmitSwap(resp.Encoded, fromAddr, toAddr, signature)
   ```

5. 查询状态
   ```go
   status, _ := bridge.GetSwapStatus(swapId)
   ```

## 命令行工具

项目包含一个命令行工具，用于快速进行跨链操作：

```bash
go run cmd/main/main.go \
  --rpc "https://rpc.merlinchain.io" \
  --key "你的私钥" \
  --amount "0.0001" \
  --from-chain "merlin" \
  --to-chain "zksync" \
  --recipient "0x接收地址(可选)"
```

## 开发测试

运行测试（需要设置环境变量）:

```bash
export RPC_URL="https://rpc.merlinchain.io"
export PRIVATE_KEY="你的私钥"
go test -v ./pkg/meson
``` 