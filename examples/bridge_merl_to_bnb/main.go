package main

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"log"
	"math/big"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

// 简化版的跨链参数结构
type SwapParams struct {
	TokenAddress common.Address
	PoolAddress  common.Address
	Amount       *big.Int
	Recipient    common.Address
	SkipApprove  bool
}

// TxData 表示交易数据
type TxData struct {
	To    common.Address
	Data  []byte
	Value *big.Int
}

func main() {
	// 从命令行参数或环境变量获取私钥
	privateKeyHex := os.Getenv("PRIVATE_KEY")
	if privateKeyHex == "" && len(os.Args) > 1 {
		privateKeyHex = os.Args[1]
	}
	if privateKeyHex == "" {
		log.Fatal("请提供私钥作为命令行参数或设置PRIVATE_KEY环境变量")
	}

	// 准备私钥
	if !strings.HasPrefix(privateKeyHex, "0x") {
		privateKeyHex = "0x" + privateKeyHex
	}
	privateKey, err := crypto.HexToECDSA(strings.TrimPrefix(privateKeyHex, "0x"))
	if err != nil {
		log.Fatalf("私钥格式错误: %v", err)
	}

	// 获取发送者地址
	publicKey := privateKey.Public().(*ecdsa.PublicKey)
	sender := crypto.PubkeyToAddress(*publicKey)
	fmt.Printf("发送者地址: %s\n", sender.Hex())

	// 解析接收地址（默认使用发送者地址）
	recipient := os.Getenv("RECIPIENT")
	recipientAddr := sender
	if recipient != "" {
		recipientAddr = common.HexToAddress(recipient)
	}
	fmt.Printf("接收者地址: %s\n", recipientAddr.Hex())

	// 解析金额（默认0.01）
	amountStr := os.Getenv("AMOUNT")
	if amountStr == "" {
		amountStr = "0.01"
	}
	// 转换金额为Wei
	amount, ok := new(big.Int).SetString(amountStr+"e18", 0)
	if !ok {
		log.Fatalf("金额格式错误: %s", amountStr)
	}
	fmt.Printf("跨链金额: %s MERL\n", amountStr)

	// 是否跳过授权
	skipApprove := os.Getenv("SKIP_APPROVE") == "true"

	// 连接到Merlin链
	rpcURL := os.Getenv("MERLIN_RPC")
	if rpcURL == "" {
		rpcURL = "https://rpc.merlinchain.io"
	}

	fmt.Println("正在连接Merlin链...")
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		log.Fatalf("连接Merlin链失败: %v", err)
	}

	// Merlin链ID和MERL代币地址
	chainID := big.NewInt(4200)
	merlTokenAddress := common.HexToAddress("0x5c46bFF4B38dc1EAE09C5BAc65872a1D8bc87378")
	poolAddress := common.HexToAddress("0x25aB3Efd52e6470681CE037cD546Dc60726948D3")

	// 设置跨链参数
	params := &SwapParams{
		TokenAddress: merlTokenAddress,
		PoolAddress:  poolAddress,
		Amount:       amount,
		Recipient:    recipientAddr,
		SkipApprove:  skipApprove,
	}

	// 1. 如果需要，授权代币给跨链桥合约
	if !params.SkipApprove {
		fmt.Println("步骤1: 授权代币给跨链桥合约...")
		approveHash, err := approveToken(client, chainID, privateKey, params.TokenAddress, params.PoolAddress, params.Amount)
		if err != nil {
			log.Fatalf("授权代币失败: %v", err)
		}
		fmt.Printf("授权交易已提交，哈希: %s\n", approveHash)
		fmt.Println("授权交易浏览器链接: https://scan.merlinchain.io/tx/" + approveHash)
	} else {
		fmt.Println("跳过授权步骤")
	}

	// 2. 执行跨链交易
	fmt.Println("步骤2: 执行Merlin到BNB链的跨链操作...")
	swapHash, err := executeCrossChainSwap(client, chainID, privateKey, params)
	if err != nil {
		log.Fatalf("跨链交易失败: %v", err)
	}

	fmt.Printf("跨链交易已发起，交易哈希: %s\n", swapHash)
	fmt.Println("跨链交易浏览器链接: https://scan.merlinchain.io/tx/" + swapHash)
	fmt.Println("请等待几分钟，然后在BNB链上检查接收地址的余额")
}

// approveToken 授权代币
func approveToken(client *ethclient.Client, chainID *big.Int, privateKey *ecdsa.PrivateKey, tokenAddr, spender common.Address, amount *big.Int) (string, error) {
	// 此处简化实现，实际应用中应该检查现有授权
	// 创建ERC20授权调用
	methodID := []byte{0x09, 0x5e, 0xa7, 0xb3} // approve(address,uint256) 函数选择器
	paddedAddress := common.LeftPadBytes(spender.Bytes(), 32)
	paddedAmount := common.LeftPadBytes(new(big.Int).Sub(new(big.Int).Exp(big.NewInt(2), big.NewInt(256), nil), big.NewInt(1)).Bytes(), 32) // 授权最大值

	data := append(methodID, append(paddedAddress, paddedAmount...)...)

	// 创建交易
	tx := &TxData{
		To:    tokenAddr,
		Data:  data,
		Value: big.NewInt(0),
	}

	// 发送交易
	return sendTransaction(client, chainID, privateKey, tx)
}

// executeCrossChainSwap 执行跨链交易
func executeCrossChainSwap(client *ethclient.Client, chainID *big.Int, privateKey *ecdsa.PrivateKey, params *SwapParams) (string, error) {
	// 此处简化实现，实际应用中应该使用Meson SDK
	// 创建跨链调用
	methodID := []byte{0x51, 0xcf, 0xb1, 0x86} // swap函数选择器（示例）

	// 简化示例 - 实际参数编码会更复杂
	paddedAmount := common.LeftPadBytes(params.Amount.Bytes(), 32)
	paddedRecipient := common.LeftPadBytes(params.Recipient.Bytes(), 32)
	paddedTokenID := common.LeftPadBytes(big.NewInt(69).Bytes(), 32) // MERL = 69

	// BNB链代码
	paddedDestChain := common.LeftPadBytes([]byte("bnb"), 32)

	data := append(methodID, append(paddedAmount, append(paddedRecipient, append(paddedTokenID, paddedDestChain...)...)...)...)

	// 创建交易
	tx := &TxData{
		To:    params.PoolAddress,
		Data:  data,
		Value: big.NewInt(0),
	}

	// 发送交易
	return sendTransaction(client, chainID, privateKey, tx)
}

// sendTransaction 发送以太坊交易
func sendTransaction(client *ethclient.Client, chainID *big.Int, privateKey *ecdsa.PrivateKey, txData *TxData) (string, error) {
	ctx := context.Background()
	from := crypto.PubkeyToAddress(privateKey.PublicKey)

	// 获取nonce
	nonce, err := client.PendingNonceAt(ctx, from)
	if err != nil {
		return "", fmt.Errorf("获取nonce失败: %w", err)
	}
	fmt.Printf("使用Nonce: %d\n", nonce)

	// 获取GasPrice
	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		return "", fmt.Errorf("获取gas价格失败: %w", err)
	}

	// 创建交易对象
	tx := types.NewTx(&types.LegacyTx{
		Nonce:    nonce,
		GasPrice: gasPrice,
		Gas:      300000, // gas limit
		To:       &txData.To,
		Value:    txData.Value,
		Data:     txData.Data,
	})

	// 签名交易
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), privateKey)
	if err != nil {
		return "", fmt.Errorf("签名交易失败: %w", err)
	}

	// 发送交易
	err = client.SendTransaction(ctx, signedTx)
	if err != nil {
		return "", fmt.Errorf("发送交易失败: %w", err)
	}

	return signedTx.Hash().Hex(), nil
}
