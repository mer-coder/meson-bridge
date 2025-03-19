package main

import (
	"context"
	"crypto/ecdsa"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/shopspring/decimal"

	"github.com/mer-coder/meson-bridge/pkg/helpers"
	"github.com/mer-coder/meson-bridge/pkg/meson"
)

func main() {
	// 命令行参数
	rpcURL := flag.String("rpc", "", "以太坊RPC URL")
	privateKeyHex := flag.String("key", "", "私钥(16进制)")
	amount := flag.String("amount", "0.0001", "跨链金额")
	fromChain := flag.String("from-chain", string(meson.ChainMerlin), "源链")
	toChain := flag.String("to-chain", string(meson.ChainZksync), "目标链")
	recipient := flag.String("recipient", "", "接收地址(默认与发送地址相同)")
	flag.Parse()

	// 检查必要参数
	if *rpcURL == "" || *privateKeyHex == "" {
		flag.Usage()
		os.Exit(1)
	}

	// 准备私钥
	privKey := *privateKeyHex
	if !strings.HasPrefix(privKey, "0x") {
		privKey = "0x" + privKey
	}

	privateKey, err := crypto.HexToECDSA(strings.TrimPrefix(privKey, "0x"))
	if err != nil {
		log.Fatalf("无效的私钥: %v", err)
	}

	// 获取地址
	publicKey := privateKey.Public().(*ecdsa.PublicKey)
	fromAddr := crypto.PubkeyToAddress(*publicKey).Hex()

	toAddr := fromAddr
	if *recipient != "" {
		toAddr = *recipient
	}

	// 解析金额
	amountDecimal, err := decimal.NewFromString(*amount)
	if err != nil {
		log.Fatalf("无效的金额: %v", err)
	}

	// 连接以太坊客户端
	client, err := ethclient.Dial(*rpcURL)
	if err != nil {
		log.Fatalf("无法连接到以太坊节点: %v", err)
	}
	defer client.Close()

	chainID, err := client.ChainID(context.Background())
	if err != nil {
		log.Fatalf("获取链ID失败: %v", err)
	}

	fmt.Printf("使用地址: %s\n", fromAddr)
	fmt.Printf("链ID: %d\n", chainID)

	// 初始化Bridge
	bridge := meson.NewBridge()

	// 1. 获取approve数据并发送approve交易
	fmt.Println("====== 步骤1: 批准合约使用代币 ======")
	approveTxData, err := bridge.GetApproveData()
	if err != nil {
		log.Fatalf("获取Approve数据失败: %v", err)
	}

	approveHash, err := helpers.SendTransaction(client, chainID, privateKey, approveTxData)
	if err != nil {
		log.Fatalf("发送Approve交易失败: %v", err)
	}
	fmt.Printf("Approve交易已发送, 哈希: %s\n", approveHash)
	fmt.Println("请等待交易确认...")

	// 2. 获取待签名消息
	fmt.Println("\n====== 步骤2: 准备跨链交易 ======")
	resp, err := bridge.BridgeMBTC(
		context.Background(),
		amountDecimal,
		fromAddr,
		toAddr,
		meson.Chain(*fromChain),
		meson.Chain(*toChain),
		meson.TokenMBTC,
		meson.TokenMBTC,
	)
	if err != nil {
		log.Fatalf("准备跨链交易失败: %v", err)
	}

	fmt.Printf("获取到跨链报价:\n")
	fmt.Printf("- 费用: %s\n", resp.PriceInfo.Fee)
	fmt.Printf("- 预计时间: %d秒\n", resp.PriceInfo.EstimatedTime)
	fmt.Printf("- 最小金额: %s\n", resp.PriceInfo.MinAmount)
	fmt.Printf("- 最大金额: %s\n", resp.PriceInfo.MaxAmount)

	// 3. 签名消息
	fmt.Println("\n====== 步骤3: 签名交易 ======")
	fmt.Printf("待签名哈希: %s\n", resp.SigningRequest.Hash)

	signature, err := signData(privateKey, common.FromHex(resp.SigningRequest.Hash))
	if err != nil {
		log.Fatalf("签名失败: %v", err)
	}
	fmt.Printf("签名完成: 0x%x\n", signature)

	// 4. 提交跨链交易
	fmt.Println("\n====== 步骤4: 提交跨链交易 ======")
	swapId, err := bridge.SubmitSwap(resp.Encoded, fromAddr, toAddr, signature)
	if err != nil {
		log.Fatalf("提交跨链交易失败: %v", err)
	}
	fmt.Printf("跨链交易已提交, ID: %s\n", swapId)

	// 5. 查询状态
	fmt.Println("\n====== 步骤5: 查询跨链状态 ======")
	status, err := bridge.GetSwapStatus(swapId)
	if err != nil {
		log.Fatalf("查询跨链状态失败: %v", err)
	}
	fmt.Printf("当前状态: %+v\n", status)
	fmt.Println("\n跨链交易已提交，请稍后使用以下命令查询状态:")
	fmt.Printf("curl -X GET \"https://relayer.meson.fi/api/v1/swap/%s\"\n", swapId)
}

// signData 签名数据
func signData(privateKey *ecdsa.PrivateKey, hash []byte) ([]byte, error) {
	// 直接对哈希进行签名
	signature, err := crypto.Sign(hash, privateKey)
	if err != nil {
		return nil, err
	}

	// 调整 v 值 (与 ethers.js 保持一致)
	signature[64] += 27

	return signature, nil
}
