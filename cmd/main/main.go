package main

import (
	"context"
	"crypto/ecdsa"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/shopspring/decimal"

	"github.com/mer-coder/meson-bridge/pkg/helpers"
	"github.com/mer-coder/meson-bridge/pkg/meson"
)

// 定义代币名称到ID的映射
var tokenNameToID = map[string]int64{
	"mbtc": 67,
	"btc":  67, // 别名
	"merl": 69,
}

// 将代币名称或ID字符串转换为代币ID
func resolveTokenID(tokenStr string) (int64, error) {
	// 尝试将字符串转换为小写
	tokenStrLower := strings.ToLower(tokenStr)

	// 检查是否在名称映射中
	if tokenID, exists := tokenNameToID[tokenStrLower]; exists {
		return tokenID, nil
	}

	// 尝试作为数字解析
	tokenID, err := strconv.ParseInt(tokenStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("无法解析代币 '%s'，既不是有效的代币名称也不是有效的ID", tokenStr)
	}

	return tokenID, nil
}

func main() {
	// 命令行参数
	rpcURL := flag.String("rpc", "", "以太坊RPC URL")
	privateKeyHex := flag.String("key", "", "私钥(16进制)")
	amount := flag.String("amount", "0.0001", "跨链金额")
	fromChain := flag.String("from-chain", string(meson.ChainMerlin), "源链")
	toChain := flag.String("to-chain", string(meson.ChainZksync), "目标链")
	recipient := flag.String("recipient", "", "接收地址(默认与发送地址相同)")
	tokenStr := flag.String("token", "", "代币ID或名称 (如 'merl' 或 '69')")
	tokenAddress := flag.String("token-address", "", "源链上代币地址(优先于token参数)")
	poolAddress := flag.String("pool-address", "", "源链上池合约地址")
	skipApprove := flag.Bool("skip-approve", false, "跳过approve步骤")
	flag.Parse()

	// 检查必要参数
	if *rpcURL == "" || *privateKeyHex == "" || *amount == "" || *tokenStr == "" || *fromChain == "" || *toChain == "" {
		fmt.Println("缺少必要参数")
		flag.Usage()
		os.Exit(1)
	}

	// 解析代币ID
	tokenID, err := resolveTokenID(*tokenStr)
	if err != nil {
		fmt.Printf("错误: %v\n", err)
		os.Exit(1)
	}

	// 获取代币名称（用于显示）
	tokenName := "Unknown"
	for name, id := range tokenNameToID {
		if id == tokenID {
			tokenName = strings.ToUpper(name)
			break
		}
	}
	if tokenName == "Unknown" {
		tokenName = fmt.Sprintf("Token(%d)", tokenID)
	}

	fmt.Printf("使用代币: %s (ID: %d)\n", tokenName, tokenID)

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
	sourceChain := meson.Chain(*fromChain)
	err = bridge.InitEthClient(*rpcURL, sourceChain)
	if err != nil {
		log.Printf("警告: 以太坊客户端初始化失败: %v", err)
		log.Println("将使用基本模式，无法检查授权状态")
	}

	// 确定使用的代币
	selectedToken := meson.Token(strconv.FormatInt(tokenID, 10))
	if selectedToken == "" {
		selectedToken = meson.TokenMBTC
	}

	// 注册自定义代币地址(如果提供)
	if *tokenAddress != "" {
		if err := bridge.RegisterTokenAddress(sourceChain, selectedToken, *tokenAddress); err != nil {
			log.Fatalf("注册代币地址失败: %v", err)
		}
		fmt.Printf("使用自定义代币地址: %s (在%s链上)\n", *tokenAddress, sourceChain)
	} else {
		// 显示使用的预设代币类型
		fmt.Printf("使用代币: %s\n", tokenName)
	}

	// 注册自定义池地址(如果提供)
	if *poolAddress != "" {
		if err := bridge.RegisterPoolAddress(sourceChain, *poolAddress); err != nil {
			log.Fatalf("注册池地址失败: %v", err)
		}
		fmt.Printf("使用自定义池地址: %s (在%s链上)\n", *poolAddress, sourceChain)
	}

	// 1. 获取approve数据并发送approve交易
	if !*skipApprove {
		fmt.Println("====== 步骤1: 批准合约使用代币 ======")

		ctx := context.Background()
		approveTxData, err := bridge.GetApproveData(ctx, fromAddr, sourceChain, selectedToken, *tokenAddress)

		if err != nil {
			log.Printf("警告: 获取Approve数据失败: %v", err)
			log.Println("跳过approve步骤。如果您尚未授权代币使用，跨链可能会失败")
		} else {
			// 发送approve交易
			approveHash, err := helpers.SendTransaction(client, chainID, privateKey, approveTxData)
			if err != nil {
				log.Fatalf("发送Approve交易失败: %v", err)
			}
			fmt.Printf("Approve交易已发送, 哈希: %s\n", approveHash)
		}
	} else {
		fmt.Println("用户选择跳过approve步骤")
	}

	// 2. 获取待签名消息
	fmt.Println("\n====== 步骤2: 准备跨链交易 ======")
	resp, err := bridge.BridgeMBTC(
		context.Background(),
		amountDecimal,
		fromAddr,
		toAddr,
		sourceChain,
		meson.Chain(*toChain),
		selectedToken,
		selectedToken,
	)
	if err != nil {
		log.Fatalf("准备跨链交易失败: %v", err)
	}

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
