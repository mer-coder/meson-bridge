package helpers

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

// TxData 表示交易数据
type TxData struct {
	To    common.Address
	Data  []byte
	Value *big.Int
}

// SendTransaction 发送以太坊交易
func SendTransaction(client *ethclient.Client, chainID *big.Int, privateKey *ecdsa.PrivateKey, txData *TxData) (string, error) {
	ctx := context.Background()
	publicKey := privateKey.Public().(*ecdsa.PublicKey)
	from := crypto.PubkeyToAddress(*publicKey)

	// 设置交易值，如果未指定则默认为0
	value := big.NewInt(0)
	if txData.Value != nil {
		value = txData.Value
	}

	// 获取Nonce - 添加日志
	pendingNonce, err := client.PendingNonceAt(ctx, from)
	if err != nil {
		return "", fmt.Errorf("获取nonce失败: %w", err)
	}
	fmt.Printf("钱包地址: %s\n", from.Hex())
	fmt.Printf("待处理Nonce: %d\n", pendingNonce)

	// 再获取一下已确认的nonce来比较（不改变逻辑，只添加日志）
	confirmedNonce, err := client.NonceAt(ctx, from, nil)
	if err != nil {
		fmt.Printf("获取已确认nonce失败: %v\n", err)
	} else {
		fmt.Printf("已确认Nonce: %d\n", confirmedNonce)
	}

	// 保持原来的逻辑
	nonce := pendingNonce

	// 获取GasPrice
	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		return "", fmt.Errorf("获取gas价格失败: %w", err)
	}
	fmt.Printf("使用的Gas价格: %s\n", gasPrice.String())

	// 创建交易对象
	tx := types.NewTx(&types.LegacyTx{
		Nonce:    nonce,
		GasPrice: gasPrice,
		Gas:      uint64(300000), // 设置一个合理的gas limit
		To:       &txData.To,
		Value:    value,
		Data:     txData.Data,
	})
	fmt.Printf("即将发送交易，使用Nonce: %d\n", nonce)
	fmt.Printf("交易目标地址: %s\n", txData.To.Hex())

	// 签名交易
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), privateKey)
	if err != nil {
		return "", fmt.Errorf("签名交易失败: %w", err)
	}

	// 发送交易
	err = client.SendTransaction(ctx, signedTx)
	if err != nil {
		fmt.Printf("发送交易失败: %v\n", err)
		return "", fmt.Errorf("发送交易失败: %w", err)
	}

	txHash := signedTx.Hash().Hex()
	fmt.Printf("交易已发送! 哈希: %s\n", txHash)
	fmt.Printf("正在等待交易确认...\n")

	// 等待交易被确认
	receipt, err := waitForTransactionReceipt(ctx, client, signedTx.Hash())
	if err != nil {
		return txHash, fmt.Errorf("等待交易确认失败: %w", err)
	}

	// 检查交易状态
	if receipt.Status == 1 {
		fmt.Printf("交易已确认成功! 区块高度: %d, Gas使用: %d\n", receipt.BlockNumber, receipt.GasUsed)
	} else {
		fmt.Printf("交易已确认但执行失败! 区块高度: %d\n", receipt.BlockNumber)
		return txHash, fmt.Errorf("交易执行失败")
	}

	return txHash, nil
}

// waitForTransactionReceipt 等待交易被确认并返回收据
func waitForTransactionReceipt(ctx context.Context, client *ethclient.Client, txHash common.Hash) (*types.Receipt, error) {
	for {
		receipt, err := client.TransactionReceipt(ctx, txHash)
		if err == nil {
			return receipt, nil
		}
		// 如果是"not found"错误，则继续等待
		if err.Error() == "not found" {
			fmt.Printf("交易仍在等待确认，继续等待...\n")
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(2 * time.Second): // 每2秒检查一次
				continue
			}
		}
		// 其他错误则直接返回
		return nil, err
	}
}
