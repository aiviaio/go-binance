package main

import (
	"context"
	"log"
	"strings"

	"github.com/aiviaio/go-binance/v2"
)

// TestSpotRecentTrades tests the /api/v3/trades endpoint
func TestSpotRecentTrades(cfg *Config) {
	log.Println("\n" + strings.Repeat("=", 60))
	log.Println("TEST: Spot Recent Trades (/api/v3/trades)")
	log.Println(strings.Repeat("=", 60))

	if cfg.SpotAPIKey == "" || cfg.SpotSecretKey == "" {
		log.Println("❌ SKIP: Spot API keys not configured")
		return
	}

	ctx := context.Background()
	client := binance.NewClient(cfg.SpotAPIKey, cfg.SpotSecretKey)

	// Test 1: Get recent trades for BTCUSDT
	log.Println("\n📊 Getting recent trades for BTCUSDT...")
	trades, err := client.NewRecentTradesService().
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

	// Test 2: Get recent trades for DOGEUSDT
	log.Println("\n📊 Getting recent trades for DOGEUSDT...")
	dogeTrades, err := client.NewRecentTradesService().
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

	log.Println("\n✅ Spot Recent Trades test completed")
}
