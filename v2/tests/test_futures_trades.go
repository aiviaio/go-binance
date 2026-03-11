package main

import (
	"context"
	"log"
	"strings"

	"github.com/aiviaio/go-binance/v2/futures"
)

// TestFuturesHistoricalTrades tests the /fapi/v1/historicalTrades endpoint
func TestFuturesHistoricalTrades(cfg *Config) {
	log.Println("\n" + strings.Repeat("=", 60))
	log.Println("TEST: Futures Historical Trades (/fapi/v1/historicalTrades)")
	log.Println(strings.Repeat("=", 60))

	if cfg.FuturesAPIKey == "" || cfg.FuturesSecretKey == "" {
		log.Println("❌ SKIP: Futures API keys not configured")
		return
	}

	ctx := context.Background()
	client := futures.NewClient(cfg.FuturesAPIKey, cfg.FuturesSecretKey)

	// Test 1: Get historical trades for BTCUSDT
	log.Println("\n📊 Getting historical trades for BTCUSDT (limit=10)...")
	trades, err := client.NewHistoricalTradesService().
		Symbol("BTCUSDT").
		Limit(10).
		Do(ctx)
	if err != nil {
		log.Printf("❌ Failed to get trades: %v", err)
		return
	}
	log.Printf("✅ Got %d trades", len(trades))
	for i, t := range trades {
		if i < 3 {
			log.Printf("   📈 ID=%d, Price=%s, Qty=%s, Time=%d", t.ID, t.Price, t.Quantity, t.Time)
		}
	}

	// Test 2: Get historical trades for DOGEUSDT
	log.Println("\n📊 Getting historical trades for DOGEUSDT (limit=5)...")
	dogeTrades, err := client.NewHistoricalTradesService().
		Symbol("DOGEUSDT").
		Limit(5).
		Do(ctx)
	if err != nil {
		log.Printf("❌ Failed to get DOGE trades: %v", err)
	} else {
		log.Printf("✅ Got %d DOGE trades", len(dogeTrades))
		for i, t := range dogeTrades {
			if i < 3 {
				log.Printf("   📈 ID=%d, Price=%s, Qty=%s", t.ID, t.Price, t.Quantity)
			}
		}
	}

	log.Println("\n✅ Futures Historical Trades test completed")
}

// TestFuturesExchangeInfo tests exchange info endpoint
func TestFuturesExchangeInfo(cfg *Config) {
	log.Println("\n" + strings.Repeat("=", 60))
	log.Println("TEST: Futures Exchange Info (/fapi/v1/exchangeInfo)")
	log.Println(strings.Repeat("=", 60))

	ctx := context.Background()
	client := futures.NewClient(cfg.FuturesAPIKey, cfg.FuturesSecretKey)

	log.Println("\n📊 Getting exchange info...")
	info, err := client.NewExchangeInfoService().Do(ctx)
	if err != nil {
		log.Printf("❌ Failed to get exchange info: %v", err)
		return
	}

	log.Printf("✅ Server time: %d", info.ServerTime)
	log.Printf("✅ Timezone: %s", info.Timezone)
	log.Printf("✅ Total symbols: %d", len(info.Symbols))

	// Show first 3 symbols
	for i, s := range info.Symbols {
		if i < 3 {
			log.Printf("   📈 %s: Status=%s, BaseAsset=%s, QuoteAsset=%s", s.Symbol, s.Status, s.BaseAsset, s.QuoteAsset)
		}
	}

	log.Println("\n✅ Futures Exchange Info test completed")
}
