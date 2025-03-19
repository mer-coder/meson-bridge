package helpers

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
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
	from := common.BytesToAddress(privateKey.PublicKey.X.Bytes())

	// 设置交易值，如果未指定则默认为0
	value := big.NewInt(0)
	if txData.Value != nil {
		value = txData.Value
	}

	// 获取Nonce
	nonce, err := client.PendingNonceAt(ctx, from)
	if err != nil {
		return "", fmt.Errorf("获取nonce失败: %w", err)
	}

	// 获取GasPrice
	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		return "", fmt.Errorf("获取gas价格失败: %w", err)
	}

	// 创建交易对象
	tx := types.NewTx(&types.LegacyTx{
		Nonce:    nonce,
		GasPrice: gasPrice,
		Gas:      uint64(300000), // 设置一个合理的gas limit
		To:       &txData.To,
		Value:    value,
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
