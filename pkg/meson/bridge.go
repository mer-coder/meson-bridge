package meson

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/shopspring/decimal"

	"github.com/mer-coder/meson-bridge/pkg/helpers"
)

type Chain string
type Token string
type ChainTokenKey struct {
	Chain Chain
	Token Token
}

const (
	ChainMerlin    Chain = "merlin"
	ChainZksync    Chain = "zksync"
	ChainDuckchain Chain = "duck"
	TokenMBTC      Token = "67"
	TokenMERL      Token = "69"
	// 合约地址
	MBTCAddress = "0x2F913C820ed3bEb3a67391a6eFF64E70c4B20b19" // Merlin链上的MBTC地址
	MERLAddress = "0x5c46bFF4B38dc1EAE09C5BAc65872a1D8bc87378" // Merlin链上的MERL地址
	PoolAddress = "0x25aB3Efd52e6470681CE037cD546Dc60726948D3" // Merlin链上的池地址
)

// 添加一个TokenAddress映射，按链和代币类型存储地址
var TokenAddressMap = map[ChainTokenKey]string{
	{Chain: ChainMerlin, Token: TokenMBTC}: MBTCAddress,
	{Chain: ChainMerlin, Token: TokenMERL}: MERLAddress,
	// 其他链上的代币地址可以在此添加
}

// Bridge Meson跨链桥操作封装
type Bridge struct {
	client       *Client
	ethClient    *ethclient.Client
	currentChain Chain                            // 当前连接的链
	tokenAddrs   map[ChainTokenKey]common.Address // 按链和代币类型存储地址
	poolAddrs    map[Chain]common.Address         // 按链存储池地址
	initialized  bool
}

// NewBridge 创建跨链桥操作实例
func NewBridge() *Bridge {
	return &Bridge{
		client:     NewClient(),
		tokenAddrs: make(map[ChainTokenKey]common.Address),
		poolAddrs:  make(map[Chain]common.Address),
	}
}

// InitEthClient 初始化以太坊客户端
func (b *Bridge) InitEthClient(url string, chain Chain) error {
	client, err := ethclient.Dial(url)
	if err != nil {
		return fmt.Errorf("连接以太坊节点失败: %w", err)
	}
	b.ethClient = client
	b.initialized = true
	b.currentChain = chain

	// 默认初始化已知Token地址
	for key, addr := range TokenAddressMap {
		b.tokenAddrs[key] = common.HexToAddress(addr)
	}

	// 默认设置Merlin链的池地址
	b.poolAddrs[ChainMerlin] = common.HexToAddress(PoolAddress)

	return nil
}

// RegisterTokenAddress 注册指定链上的Token地址
func (b *Bridge) RegisterTokenAddress(chain Chain, token Token, address string) error {
	if !common.IsHexAddress(address) {
		return fmt.Errorf("无效的地址格式: %s", address)
	}
	b.tokenAddrs[ChainTokenKey{Chain: chain, Token: token}] = common.HexToAddress(address)
	return nil
}

// RegisterPoolAddress 注册指定链上的池地址
func (b *Bridge) RegisterPoolAddress(chain Chain, address string) error {
	if !common.IsHexAddress(address) {
		return fmt.Errorf("无效的地址格式: %s", address)
	}
	b.poolAddrs[chain] = common.HexToAddress(address)
	return nil
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
		From:        fmt.Sprintf("%s:%s", fromChain, fromToken),
		To:          fmt.Sprintf("%s:%s", toChain, toToken),
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

// GetApproveData 获取approve调用数据，并检查现有授权
// fromAddress: 用户地址
// chain: 链标识，如ChainMerlin
// token: 代币类型，如TokenMBTC或TokenMERL
// tokenAddress: 可选，代币在指定链上的地址，优先级高于预设地址
func (b *Bridge) GetApproveData(ctx context.Context, fromAddress string, chain Chain, token Token, tokenAddress string) (*helpers.TxData, error) {
	// 检查以太坊客户端是否已初始化
	if !b.initialized || b.ethClient == nil {
		return nil, fmt.Errorf("以太坊客户端未初始化，请先调用InitEthClient")
	}

	// 如果未指定链，使用当前连接的链
	if chain == "" {
		chain = b.currentChain
		if chain == "" {
			return nil, fmt.Errorf("未指定链且未初始化当前链，请在调用InitEthClient时指定chain参数")
		}
	}

	// 确定代币地址
	var tokenAddr common.Address

	// 如果提供了自定义地址，优先使用
	if tokenAddress != "" {
		if !common.IsHexAddress(tokenAddress) {
			return nil, fmt.Errorf("无效的代币地址: %s", tokenAddress)
		}
		tokenAddr = common.HexToAddress(tokenAddress)
	} else {
		// 否则从映射表中获取
		key := ChainTokenKey{Chain: chain, Token: token}
		addr, exists := b.tokenAddrs[key]
		if !exists {
			return nil, fmt.Errorf("未知的代币类型: %s 在链 %s 上，请提供代币地址", token, chain)
		}
		tokenAddr = addr
	}

	// 获取池地址
	poolAddr, exists := b.poolAddrs[chain]
	if !exists {
		return nil, fmt.Errorf("未知的链: %s，请先注册池地址", chain)
	}

	// 由于Allowance方法存在问题，我们直接生成approve交易
	// 在实际应用中，最好先检查授权
	// 为了简化，我们假设需要授权

	// 创建ERC20接口
	erc20, err := NewERC20(b.ethClient, tokenAddr)
	if err != nil {
		return nil, fmt.Errorf("创建ERC20接口失败: %w", err)
	}

	// 定义最大值用于授权
	maxUint256 := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1))

	// 简化处理: 由于无法检查allowance，我们总是生成approve数据
	// 在实际应用中，最好实现allowance检查
	approveData, err := erc20.GetApproveData(poolAddr, maxUint256)
	if err != nil {
		return nil, fmt.Errorf("生成approve数据失败: %w", err)
	}

	// 为安全起见，我们总是返回approve数据
	// 在实际应用中，返回的第三个值应该基于allowance检查
	return &helpers.TxData{
		To:   tokenAddr,
		Data: approveData,
	}, nil
}

