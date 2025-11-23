package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const (
	binanceFuturesBaseURL = "https://fapi.binance.com"
)

type BinanceClient struct {
	apiKey     string
	secretKey  string
	httpClient *http.Client
}

func NewBinanceClient(apiKey, secretKey string) *BinanceClient {
	return &BinanceClient{
		apiKey:     apiKey,
		secretKey:  secretKey,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *BinanceClient) sign(query string) string {
	mac := hmac.New(sha256.New, []byte(c.secretKey))
	mac.Write([]byte(query))
	return hex.EncodeToString(mac.Sum(nil))
}

type OrderRequest struct {
	Symbol      string
	Side        string
	Type        string
	Quantity    string
	Price       string
	TimeInForce string
	PriceMatch  string
}

type OrderResponse struct {
	OrderID       int64  `json:"orderId"`
	Symbol        string `json:"symbol"`
	Status        string `json:"status"`
	ClientOrderID string `json:"clientOrderId"`
	Price         string `json:"price"`
	AvgPrice      string `json:"avgPrice"`
	OrigQty       string `json:"origQty"`
	ExecutedQty   string `json:"executedQty"`
	Type          string `json:"type"`
	Side          string `json:"side"`
	TimeInForce   string `json:"timeInForce"`
	PriceMatch    string `json:"priceMatch"`
}

func (c *BinanceClient) CreateOrder(req OrderRequest) (*OrderResponse, error) {
	params := url.Values{}
	params.Set("symbol", req.Symbol)
	params.Set("side", req.Side)
	params.Set("type", req.Type)

	if req.Quantity != "" {
		params.Set("quantity", req.Quantity)
	}

	if req.Type == "LIMIT" {
		if req.TimeInForce == "" {
			req.TimeInForce = "GTC"
		}
		params.Set("timeInForce", req.TimeInForce)

		if req.PriceMatch != "" {
			params.Set("priceMatch", req.PriceMatch)
		} else if req.Price != "" {
			params.Set("price", req.Price)
		}
	}

	params.Set("timestamp", strconv.FormatInt(time.Now().UnixMilli(), 10))

	queryString := params.Encode()
	signature := c.sign(queryString)
	queryString += "&signature=" + signature

	endpoint := binanceFuturesBaseURL + "/fapi/v1/order?" + queryString

	httpReq, err := http.NewRequest("POST", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("request creation failed: %w", err)
	}

	httpReq.Header.Set("X-MBX-APIKEY", c.apiKey)
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("response read failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("binance error (status %d): %s", resp.StatusCode, string(body))
	}

	var orderResp OrderResponse
	if err := json.Unmarshal(body, &orderResp); err != nil {
		return nil, fmt.Errorf("response parse failed: %w", err)
	}

	return &orderResp, nil
}
