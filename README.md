tv-binance-bot
==============

Requirements
------------
- Go 1.23+ (or Docker)
- Binance Futures API access
- Ability to send webhooks from TradingView strategy

Environment Variables
---------------------
- `BINANCE_API_KEY`, `BINANCE_SECRET_KEY`
- `WEBHOOK_PASSPHRASE`
- `ENABLE_LIVE_ORDERS` (optional, sends orders when `true`)
- `COALESCE_DELAY_MS` (optional, same-bar filter delay)
- `PORT` (optional, default 8080)

Local Development
-----------------
1. Export required environment variables in your shell:
```bash
export BINANCE_API_KEY="your_api_key"
export BINANCE_SECRET_KEY="your_secret_key"
export WEBHOOK_PASSPHRASE="your_passphrase"
export ENABLE_LIVE_ORDERS="true"
```
2. `go run main.go`

Docker
------
1. `docker build -t tv-binance-bot .`
2. `docker run --rm -p 8080:8080 \
   -e BINANCE_API_KEY="your_api_key" \
   -e BINANCE_SECRET_KEY="your_secret_key" \
   -e WEBHOOK_PASSPHRASE="your_passphrase" \
   -e ENABLE_LIVE_ORDERS="true" \
   tv-binance-bot`

TradingView Alert Payload
-------------------------
**Market Order:**
```json
{
  "passphrase": "YOUR_WEBHOOK_PASSPHRASE",
  "symbol": "{{ticker}}",
  "side": "{{strategy.order.action}}",
  "type": "MARKET",
  "quantity": "0.1",
  "bar_time": "{{time}}",
  "order_id": "{{strategy.order.id}}"
}
```

**Limit Order:**
```json
{
  "passphrase": "YOUR_WEBHOOK_PASSPHRASE",
  "symbol": "{{ticker}}",
  "side": "{{strategy.order.action}}",
  "type": "LIMIT",
  "quantity": "0.1",
  "price": "{{close}}",
  "time_in_force": "GTC",
  "bar_time": {{time}},
  "order_id": "{{strategy.order.id}}"
}
```

**BBO Order (Binance priceMatch OPPONENT):**
```json
{
  "passphrase": "YOUR_WEBHOOK_PASSPHRASE",
  "symbol": "{{ticker}}",
  "side": "{{strategy.order.action}}",
  "type": "BBO",
  "quantity": "0.1",
  "price_match": "OPPONENT",
  "bar_time": {{time}},
  "order_id": "{{strategy.order.id}}"
}
```

`price_match` options: `OPPONENT`, `OPPONENT_5`, `OPPONENT_10`, `OPPONENT_20`, `QUEUE`, `QUEUE_5`, `QUEUE_10`, `QUEUE_20`

