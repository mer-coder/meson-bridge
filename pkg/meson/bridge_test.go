package meson

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/mer-coder/meson-bridge/pkg/helpers"
)

type EthSigner struct {
	privateKey *ecdsa.PrivateKey
}

func NewEthSigner(privateKeyHex string) (*EthSigner, error) {
	privateKey, err := crypto.HexToECDSA(privateKeyHex[2:]) // 移除"0x"前缀
	if err != nil {
		return nil, err
	}
	return &EthSigner{privateKey: privateKey}, nil
}

func (s *EthSigner) Sign(message []byte) ([]byte, error) {
	// 计算消息哈希
	msgHash := crypto.Keccak256Hash(message).Bytes()

	// 添加以太坊签名前缀
	prefixedHash := crypto.Keccak256Hash([]byte(fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(msgHash), msgHash))).Bytes()

	// 签名
	signature, err := crypto.Sign(prefixedHash, s.privateKey)
	if err != nil {
		return nil, err
	}

	// 调整 v 值
	signature[64] += 27

	return signature, nil
}

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

func TestBridge_Bridge(t *testing.T) {
	// 从环境变量获取配置
	rpcURL := os.Getenv("RPC_URL")
	privateKeyHex := os.Getenv("PRIVATE_KEY")
	if rpcURL == "" || privateKeyHex == "" {
		t.Skip("请设置RPC_URL和PRIVATE_KEY环境变量")
	}


	privateKey, err := crypto.HexToECDSA(strings.TrimPrefix(privateKeyHex, "0x"))
	require.NoError(t, err)
	publicKey := privateKey.Public().(*ecdsa.PublicKey)
	fromAddr := crypto.PubkeyToAddress(*publicKey).Hex()
	toAddr := fromAddr

	
	// 初始化Bridge
	bridge := NewBridge()

	bridge.InitEthClient(rpcURL, ChainMerlin)
	// 1. 获取approve数据并发送approve交易
	approveTxData, err := bridge.GetApproveData(context.Background(), fromAddr, ChainMerlin, TokenMERL, "")
	require.NoError(t, err)

	chainID, err := bridge.ethClient.ChainID(context.Background())
	require.NoError(t, err)

	approveHash, err := helpers.SendTransaction(bridge.ethClient, chainID, privateKey, approveTxData)
	require.NoError(t, err)
	t.Logf("Approve tx hash: %s", approveHash)

	// 2. 获取待签名消息
	amount := decimal.NewFromFloat(6)
	resp, err := bridge.BridgeMBTC(context.Background(), amount, fromAddr, toAddr, ChainMerlin, "bnb", TokenMERL, TokenMERL)
	require.NoError(t, err)
	require.NotEmpty(t, resp)

	// 3. 直接对哈希进行签名
	signature, err := signData(privateKey, common.FromHex(resp.SigningRequest.Hash))
	require.NoError(t, err)
	require.NotEmpty(t, signature)
	t.Logf("Signature: 0x%s", hexutil.Encode(signature))

	// 4. 提交跨链交易
	swapId, err := bridge.SubmitSwap(resp.Encoded, fromAddr, toAddr, signature)
	require.NoError(t, err)
	require.NotEmpty(t, swapId)
	t.Logf("Swap ID: %s", swapId)

	// 5. 持续查询并打印状态
	for i := 0; ; i++ {
		status, err := bridge.GetSwapStatus(swapId)
		require.NoError(t, err)
		require.NotNil(t, status)
		fmt.Printf("Swap Status: %+v\n", status)

		// 如果状态为完成或失败，则退出循环
		if status["expire"] != nil || status["RELEASED"] != nil || status["CANCELLED"] != nil {
			break
		}

		// 等待一段时间后再次查询
		time.Sleep(1 * time.Second)
	}
}
