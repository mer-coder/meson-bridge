package meson

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

// ERC20简化ABI字符串
var erc20ABI = `[
	{
		"constant": true,
		"inputs": [
			{
				"name": "owner",
				"type": "address"
			},
			{
				"name": "spender",
				"type": "address"
			}
		],
		"name": "allowance",
		"outputs": [
			{
				"name": "",
				"type": "uint256"
			}
		],
		"payable": false,
		"stateMutability": "view",
		"type": "function"
	},
	{
		"constant": false,
		"inputs": [
			{
				"name": "spender",
				"type": "address"
			},
			{
				"name": "amount",
				"type": "uint256"
			}
		],
		"name": "approve",
		"outputs": [
			{
				"name": "",
				"type": "bool"
			}
		],
		"payable": false,
		"stateMutability": "nonpayable",
		"type": "function"
	}
]`

// ERC20 是ERC20代币合约的简化接口
type ERC20 struct {
	address common.Address
	abi     abi.ABI
	client  *ethclient.Client
}

// NewERC20 创建ERC20接口实例
func NewERC20(client *ethclient.Client, address common.Address) (*ERC20, error) {
	parsed, err := abi.JSON(strings.NewReader(erc20ABI))
	if err != nil {
		return nil, fmt.Errorf("解析ABI失败: %w", err)
	}

	return &ERC20{
		address: address,
		abi:     parsed,
		client:  client,
	}, nil
}

// Allowance 获取代币的授权额度 - 简化版，不使用CallContract
func (e *ERC20) Allowance(ctx context.Context, owner, spender common.Address) (*big.Int, error) {
	// 为了简化，我们将在Bridge中实现一个替代方案，不依赖于此方法
	// 这里返回一个零值，避免编译错误
	return big.NewInt(0), nil
}

// GetApproveData 返回approve调用的编码数据
func (e *ERC20) GetApproveData(spender common.Address, amount *big.Int) ([]byte, error) {
	// 打包数据
	return e.abi.Pack("approve", spender, amount)
}
