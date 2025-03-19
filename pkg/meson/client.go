package meson

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const (
	baseURL = "https://relayer.meson.fi/api/v1"
)

// Client Meson API客户端封装
type Client struct {
	httpClient *http.Client
}

// NewClient 创建API客户端实例
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{},
	}
}

// GetPrice 获取跨链费用
func (c *Client) GetPrice(req *PriceRequest) (*PriceResponse, error) {
	return doRequest[PriceResponse](c, "POST", "/price", req)
}

// EncodeSwap 编码跨链交易
func (c *Client) EncodeSwap(req *SwapEncodeRequest) (*SwapEncodeResponse, error) {
	return doRequest[SwapEncodeResponse](c, "POST", "/swap", req)
}

// SubmitSwap 提交跨链交易
func (c *Client) SubmitSwap(encoded string, req *SwapSubmitRequest) (*SwapResponse, error) {
	return doRequest[SwapResponse](c, "POST", fmt.Sprintf("/swap/%s", encoded), req)
}

// GetSwapStatus 获取跨链状态
func (c *Client) GetSwapStatus(swapId string) (map[string]any, error) {
	result, err := doRequest[map[string]any](c, "GET", fmt.Sprintf("/swap/%s", swapId), nil)
	if err != nil {
		return nil, err
	}
	return *result, nil
}

// doRequest 通用请求处理
func doRequest[T any](c *Client, method, path string, body interface{}) (*T, error) {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("序列化请求失败: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, baseURL+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API请求失败,状态码:%d,响应:%s", resp.StatusCode, string(respBody))
	}

	var apiResp struct {
		Result T `json:"result"`
	}
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}
	result := apiResp.Result

	return &result, nil
}
