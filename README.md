# Meson 跨链桥 Go SDK

这是一个用于与 Meson 跨链桥交互的 Go 语言 SDK，可以帮助开发者轻松实现跨链转账功能。

## 功能特性

- 获取跨链转账费用
- 编码和签名跨链交易
- 提交跨链交易
- 查询跨链交易状态
- 生成必要的 ERC20 代币授权交易
- 支持多种代币(MBTC, MERL)和多条链

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
    
    fmt.Printf("跨链费用: %s\n", price)
}
```

## 完整跨链流程

1. 初始化以太坊客户端（指定当前连接的链）
   ```go
   // 初始化以太坊客户端，指定当前链为Merlin
   bridge.InitEthClient("https://rpc.merlinchain.io", meson.ChainMerlin)
   ```

2. (可选)注册代币和池地址（如果非预设代币）
   ```go
   // 注册ZKsync链上的代币999地址
   bridge.RegisterTokenAddress(meson.ChainZksync, "999", "0x123...")
   
   // 注册bnb链上的池地址
   bridge.RegisterPoolAddress("bnb", "0x456...")
   ```

3. (可选)授权跨链桥合约使用你的代币
   ```go
   // 检查并授权Merlin链上的MBTC
   txData, err := bridge.GetApproveData(
       context.Background(),
       "0xYourAddress",
       meson.ChainMerlin,   // 指定链
       meson.TokenMBTC,     // 代币类型
       ""                   // 可选，自定义代币地址
   )
   
   if err != nil {
       fmt.Printf("获取授权数据失败: %v\n", err)
   } else {
       // 发送授权交易
       approveHash, err := helpers.SendTransaction(client, chainID, privateKey, txData)
   }
   ```

4. 准备跨链交易
   ```go
   resp, _ := bridge.BridgeMBTC(
       context.Background(),
       decimal.NewFromFloat(0.0001),
       "0xFromAddress",
       "0xToAddress",
       meson.ChainMerlin,   // 源链
       meson.ChainZksync,   // 目标链
       meson.TokenMBTC,     // 源链代币
       meson.TokenMBTC,     // 目标链代币
   )
   ```

5. 签名消息
   ```go
   signature, _ := signData(privateKey, common.FromHex(resp.SigningRequest.Hash))
   ```

6. 提交跨链交易
   ```go
   swapId, _ := bridge.SubmitSwap(resp.Encoded, fromAddr, toAddr, signature)
   ```

7. 查询状态
   ```go
   status, _ := bridge.GetSwapStatus(swapId)
   ```

## 预设代币地址

SDK预设了以下Merlin链上的代币地址：

- MBTC (TokenMBTC): `0x2F913C820ed3bEb3a67391a6eFF64E70c4B20b19`
- MERL (TokenMERL): `0x5c46bFF4B38dc1EAE09C5BAc65872a1D8bc87378`

对于其他链上的代币，需要通过以下方式注册：

```go
// 例如注册BNB链上的MBTC地址
bridge.RegisterTokenAddress(meson.Chain("bnb"), meson.TokenMBTC, "0x...")
```

## 命令行工具

项目包含一个命令行工具，用于快速进行跨链操作：

```bash
go run cmd/main/main.go \
  --rpc https://rpc.merlinchain.io \
  --key 你的私钥 \
  --amount 0.0001 \
  --from-chain merlin \
  --to-chain zksync \
  --token merl \          # 支持代币名称(merl, mbtc)或ID(69=MERL, 67=MBTC)
  --token-address 0x... \ # 可选，源链上的代币地址
  --pool-address 0x... \  # 可选，源链上的池合约地址
  --recipient 0x接收地址 \  # 可选，默认使用发送者地址
  --skip-approve          # 可选，跳过授权步骤
```

## 授权逻辑说明

SDK在处理代币授权时遵循以下逻辑：

1. 必须先初始化以太坊客户端并指定当前链
2. 检查指定链上代币对池合约的已有授权
3. 如果已有授权足够（大于2^255），则不进行新的授权
4. 如果授权不足，则生成授权最大值(2^256-1)的交易数据
5. 如果未提供代币地址且该链上没有预设的代币地址，会返回错误
6. 同样，如果该链上没有预设的池地址且未提供，也会返回错误
7. 当`--skip-approve`设置为true时，会跳过授权步骤

## 多链支持

SDK支持在不同链上操作不同的代币：

1. 每个链上的代币地址可能不同，需要分别注册
2. 每个链上的池合约地址也可能不同，同样需要注册
3. Merlin链上的MBTC、MERL和池地址已预设
4. 对于非Merlin链，使用前必须注册相应的代币和池地址

## 开发测试

运行MERL到BNB跨链测试（需要设置环境变量）:

```bash
export RPC_URL="https://rpc.merlinchain.io"
export PRIVATE_KEY="你的私钥"
go test -v ./pkg/meson
```

### 命令行测试

#### MERL到BNB跨链命令行示例

```bash
go run cmd/main/main.go \
  --rpc https://rpc.merlinchain.io \
  --key 您的私钥 \
  --amount 6 \
  --token merl \
  --from-chain merlin \
  --to-chain bnb \
```

如果您已经对MERL代币进行过授权，可以使用`--skip-approve`参数跳过授权步骤：

```bash
go run cmd/main/main.go \
  --rpc https://rpc.merlinchain.io \
  --key 您的私钥 \
  --amount 6 \
  --token merl \
  --from-chain merlin \
  --to-chain bnb \
  --skip-approve
```

### 更多信息

有关更多详细信息和高级用法，请联系作者。 