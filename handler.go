package main

import (
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
)

type WebhookPayload struct {
	Passphrase  FlexString `json:"passphrase"`
	Symbol      FlexString `json:"symbol"`
	Side        FlexString `json:"side"`
	Action      FlexString `json:"action,omitempty"`
	Type        FlexString `json:"type"`
	Quantity    FlexString `json:"quantity"`
	BarTime     FlexString `json:"bar_time,omitempty"`
	OrderID     FlexString `json:"order_id,omitempty"`
	Price       FlexString `json:"price,omitempty"`
	TimeInForce FlexString `json:"time_in_force,omitempty"`
	PriceMatch  FlexString `json:"price_match,omitempty"`
}

type pendingKey struct {
	Symbol    string
	BarMinute int64
}

type pendingSignal struct {
	Side    string
	Timer   *time.Timer
	Payload WebhookPayload
}

var (
	pendingMu    sync.Mutex
	pendingByKey = make(map[pendingKey]*pendingSignal)
	defaultDelay = func() time.Duration {
		if v := os.Getenv("COALESCE_DELAY_MS"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n >= 0 {
				return time.Duration(n) * time.Millisecond
			}
		}
		return 1500 * time.Millisecond
	}()
)

func resolveSide(p WebhookPayload) (string, bool) {
	action := strings.ToUpper(strings.TrimSpace(p.Action.String()))
	side := strings.ToUpper(strings.TrimSpace(p.Side.String()))

	if action == "OPEN LONG" || action == "OPENLONG" || action == "OPEN_LONG" {
		return "BUY", false
	}
	if action == "CLOSE LONG" || action == "CLOSELONG" || action == "CLOSE_LONG" {
		return "SELL", true
	}
	if action == "OPEN SHORT" || action == "OPENSHORT" || action == "OPEN_SHORT" {
		return "SELL", false
	}
	if action == "CLOSE SHORT" || action == "CLOSESHORT" || action == "CLOSE_SHORT" {
		return "BUY", true
	}

	if side == "BUY" || side == "SELL" {
		return side, false
	}

	return "BUY", false
}

func barMinuteFromMs(ms int64) int64 {
	if ms <= 0 {
		nowMs := time.Now().UnixMilli()
		return nowMs / 60000
	}
	return ms / 60000
}

func executeOrder(client *BinanceClient, p WebhookPayload, side string, reduceOnly bool) error {
	orderType := strings.ToUpper(p.Type.String())

	reduceOnlyStr := ""
	if reduceOnly {
		reduceOnlyStr = "true"
	}

	req := OrderRequest{
		Symbol:      p.Symbol.String(),
		Side:        side,
		Type:        orderType,
		Quantity:    p.Quantity.String(),
		Price:       p.Price.String(),
		TimeInForce: p.TimeInForce.String(),
		PriceMatch:  p.PriceMatch.String(),
		ReduceOnly:  reduceOnlyStr,
	}

	if orderType == "BBO" {
		req.Type = "LIMIT"
		if req.PriceMatch == "" {
			req.PriceMatch = "OPPONENT"
		}
	}

	order, err := client.CreateOrder(req)
	if err != nil {
		return err
	}

	log.Printf("Order Successful! Order ID: %d - %s %s Q:%s Type:%s Price:%s",
		order.OrderID, side, p.Symbol.String(), p.Quantity.String(), order.Type, order.Price)
	return nil
}

func Handler(binanceClient *BinanceClient, passphrase string, enableLiveOrders bool) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		var p WebhookPayload
		if err := Json.Unmarshal(c.Body(), &p); err != nil {
			log.Println("JSON error:", err)
			return c.Status(fiber.StatusBadRequest).SendString("JSON decode error")
		}

		if p.Passphrase.String() != passphrase {
			log.Println("Unauthorized access attempt!")
			return c.Status(fiber.StatusUnauthorized).SendString("Unauthorized")
		}

		side, reduceOnly := resolveSide(p)

		barMinute := barMinuteFromMs(p.BarTime.Int64())
		key := pendingKey{Symbol: p.Symbol.String(), BarMinute: barMinute}

		pendingMu.Lock()
		if existing, ok := pendingByKey[key]; ok {
			if existing.Side != side {
				if existing.Timer.Stop() {
					delete(pendingByKey, key)
					pendingMu.Unlock()
					log.Printf("Opposite signals detected within same bar (%s), both cancelled (%s)", p.Symbol, time.UnixMilli(p.BarTime.Int64()).UTC())
					return c.SendString("Signals coalesced and dropped for same bar")
				}
				pendingMu.Unlock()
				return c.SendString("Late opposite signal ignored; previous executed")
			}
			pendingMu.Unlock()
			return c.SendString("Duplicate signal ignored")
		}

		delay := defaultDelay
		log.Printf("Signal received and queued: %s %s Q:%s BarMinute:%d OrderID:%s", side, p.Symbol, p.Quantity, barMinute, p.OrderID)
		timer := time.AfterFunc(delay, func() {
			log.Printf("Order Details | Symbol:%s Side:%s Type:%s Qty:%s OrderID:%s BarTime:%d",
				p.Symbol, side, p.Type, p.Quantity, p.OrderID, p.BarTime)
			if enableLiveOrders {
				if err := executeOrder(binanceClient, p, side, reduceOnly); err != nil {
					log.Printf("Binance Error: %v", err)
				}
			} else {
				log.Printf("LOG-ONLY mode active; order not sent to Binance (%s %s).", side, p.Symbol)
			}
			pendingMu.Lock()
			delete(pendingByKey, key)
			pendingMu.Unlock()
		})
		pendingByKey[key] = &pendingSignal{
			Side:    side,
			Timer:   timer,
			Payload: p,
		}
		pendingMu.Unlock()

		return c.Status(fiber.StatusAccepted).SendString("Signal queued (coalescing window)")
	}
}
