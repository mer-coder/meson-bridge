package meson

// PriceRequest 获取跨链费用请求
type PriceRequest struct {
	From        string `json:"from"` // 格式: chain:token
	To          string `json:"to"`   // 格式: chain:token
	Amount      string `json:"amount"`
	FromAddress string `json:"fromAddress"`
}

// PriceResponse 跨链费用响应
type PriceResponse struct {
	Fee           string `json:"fee"`
	EstimatedTime int    `json:"estimatedTime"`
	MinAmount     string `json:"minAmount"`
	MaxAmount     string `json:"maxAmount"`
}

// SwapEncodeRequest 编码跨链交易请求
type SwapEncodeRequest struct {
	From        string `json:"from"`
	To          string `json:"to"`
	Amount      string `json:"amount"`
	FromAddress string `json:"fromAddress"`
	Recipient   string `json:"recipient"`
	ExpireTs    int64  `json:"expireTs"`
}

// SwapEncodeResponse 编码跨链交易响应
type SwapEncodeResponse struct {
	Encoded        string `json:"encoded"`
	SigningRequest struct {
		Message string `json:"message"`
		Hash    string `json:"hash"`
	} `json:"signingRequest"`
	PriceInfo PriceResponse `json:"priceInfo"`
}

// SwapSubmitRequest 提交跨链交易请求
type SwapSubmitRequest struct {
	FromAddress string `json:"fromAddress"`
	Recipient   string `json:"recipient"`
	Signature   string `json:"signature"`
}

// SwapResponse 跨链交易响应
type SwapResponse struct {
	SwapId string `json:"swapId"`
}

// SwapStatus 跨链交易状态
type SwapStatus struct {
	Status    string `json:"status"`
	FromTx    string `json:"fromTx"`
	ToTx      string `json:"toTx"`
	Timestamp int64  `json:"timestamp"`
}
