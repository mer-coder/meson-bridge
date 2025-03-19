package meson

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/shopspring/decimal"

	"github.com/mer-coder/meson-bridge/pkg/helpers"
)

type Chain string
type Token string

const (
	ChainMerlin    Chain = "merlin"
	ChainZksync    Chain = "zksync"
	ChainDuckchain Chain = "duck"
	TokenMBTC      Token = "67"
	TokenMERL      Token = "69"
	// 合约地址
	MBTCAddress = "0x2F913C820ed3bEb3a67391a6eFF64E70c4B20b19"
	PoolAddress = "0x25aB3Efd52e6470681CE037cD546Dc60726948D3"
)

// Bridge Meson跨链桥操作封装
type Bridge struct {
	client *Client
}

// NewBridge 创建跨链桥操作实例
func NewBridge() *Bridge {
	return &Bridge{
		client: NewClient(),
	}
}

// BridgeMBTC 从Merlin跨链MBTC到Linea
func (b *Bridge) BridgeMBTC(ctx context.Context, amount decimal.Decimal, fromAddr, toAddr string, fromChain, toChain Chain, fromToken, toToken Token) (*SwapEncodeResponse, error) {
	// 验证参数
	if err := b.validateAddresses(fromAddr, toAddr); err != nil {
		return nil, err
	}

	if fromChain == "" {
		fromChain = ChainMerlin
	}
	if toChain == "" {
		toChain = ChainZksync
	}
	if fromToken == "" {
		fromToken = "67"
	}
	if toToken == "" {
		toToken = "67"
	}

	// 编码跨链交易
	encodeResp, err := b.client.EncodeSwap(&SwapEncodeRequest{
		From:        fmt.Sprintf("%s:67", fromChain),
		To:          fmt.Sprintf("%s:67", toChain),
		Amount:      amount.String(),
		FromAddress: fromAddr,
		Recipient:   toAddr,
		ExpireTs:    time.Now().Add(time.Minute * 110).Unix(), // 返回的是Unix时间戳(秒数),例如1704074400表示2024-01-01 02:00:00 UTC
	})
	if err != nil {
		return nil, fmt.Errorf("编码交易失败: %w", err)
	}

	return encodeResp, nil
}

// validateAddresses 验证地址格式
func (b *Bridge) validateAddresses(addresses ...string) error {
	for _, addr := range addresses {
		if !common.IsHexAddress(addr) {
			return fmt.Errorf("无效的地址格式: %s", addr)
		}
	}
	return nil
}

// SubmitSwap 提交跨链交易
func (b *Bridge) SubmitSwap(encoded, fromAddr, toAddr string, sig []byte) (string, error) {
	submitResp, err := b.client.SubmitSwap(encoded, &SwapSubmitRequest{
		FromAddress: fromAddr,
		Recipient:   toAddr,
		Signature:   "0x" + common.Bytes2Hex(sig),
	})
	if err != nil {
		return "", fmt.Errorf("提交交易失败: %w", err)
	}

	return submitResp.SwapId, nil
}

// GetSwapStatus 获取跨链状态
func (b *Bridge) GetSwapStatus(swapId string) (map[string]any, error) {
	return b.client.GetSwapStatus(swapId)
}

// GetApproveData 获取approve调用数据
func (b *Bridge) GetApproveData() (*helpers.TxData, error) {
	// ERC20 approve方法的ABI
	approveABI := `[{"constant":false,"inputs":[{"name":"spender","type":"address"},{"name":"amount","type":"uint256"}],"name":"approve","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"}]`

	parsedABI, err := abi.JSON(strings.NewReader(approveABI))
	if err != nil {
		return nil, fmt.Errorf("解析ABI失败: %w", err)
	}

	maxUint256 := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1))
	input, err := parsedABI.Pack("approve", common.HexToAddress(PoolAddress), maxUint256)
	if err != nil {
		return nil, fmt.Errorf("打包交易数据失败: %w", err)
	}

	return &helpers.TxData{
		To:   common.HexToAddress(MBTCAddress),
		Data: input,
	}, nil
}
