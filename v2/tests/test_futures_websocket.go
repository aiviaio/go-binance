package main

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/aiviaio/go-binance/v2/futures"
)

// TestFuturesUserDataStream tests Futures user data stream with new /private endpoint
func TestFuturesUserDataStream(cfg *Config) {
	log.Println("\n" + strings.Repeat("=", 60))
	log.Println("TEST: Futures User Data Stream (/private endpoint)")
	log.Println(strings.Repeat("=", 60))

	if cfg.FuturesAPIKey == "" || cfg.FuturesSecretKey == "" {
		log.Println("❌ SKIP: Futures API keys not configured")
		return
	}

	ctx := context.Background()
	client := futures.NewClient(cfg.FuturesAPIKey, cfg.FuturesSecretKey)

	// Get listen key
	log.Println("🔑 Getting listen key...")
	listenKey, err := client.NewStartUserStreamService().Do(ctx)
	if err != nil {
		log.Printf("❌ Failed to get listen key: %v", err)
		return
	}
	log.Printf("✅ Listen key: %s...", listenKey[:20])

	// Connect to WebSocket
	log.Println("📡 Connecting to WebSocket (new /private endpoint)...")
	wsHandler := func(event *futures.WsUserDataEvent) {
		log.Printf("📨 Event: %s", event.Event)
		if event.AccountUpdate.Reason != "" {
			log.Printf("   💰 Account Update: %s", event.AccountUpdate.Reason)
		}
		if event.OrderTradeUpdate.Symbol != "" {
			log.Printf("   📋 Order Update: %s %s %s", event.OrderTradeUpdate.Symbol, event.OrderTradeUpdate.Side, event.OrderTradeUpdate.Status)
		}
	}
	errHandler := func(err error) {
		log.Printf("❌ WebSocket error: %v", err)
	}

	doneC, stopC, err := futures.WsUserDataServe(listenKey, wsHandler, errHandler)
	if err != nil {
		log.Printf("❌ Failed to connect: %v", err)
		return
	}
	log.Println("✅ Connected to Futures user data stream!")

	// Listen for 10 seconds
	log.Println("👂 Listening for events (10 seconds)...")
	go func() {
		time.Sleep(10 * time.Second)
		close(stopC)
	}()

	<-doneC
	log.Println("✅ Futures User Data Stream test completed")
}

// TestFuturesMarketStream tests Futures market data with new /market endpoint
func TestFuturesMarketStream(cfg *Config) {
	log.Println("\n" + strings.Repeat("=", 60))
	log.Println("TEST: Futures Market Data Stream (/market endpoint)")
	log.Println(strings.Repeat("=", 60))

	// Test mini ticker
	log.Println("📡 Connecting to BTCUSDT mini ticker...")
	
	eventCount := 0
	handler := func(event *futures.WsMiniMarketTickerEvent) {
		eventCount++
		if eventCount <= 5 {
			log.Printf("📈 %s Price: %s, Volume: %s", event.Symbol, event.ClosePrice, event.Volume)
		}
	}
	errHandler := func(err error) {
		log.Printf("❌ Error: %v", err)
	}

	doneC, stopC, err := futures.WsMiniMarketTickerServe("BTCUSDT", handler, errHandler)
	if err != nil {
		log.Printf("❌ Failed to connect: %v", err)
		return
	}
	log.Println("✅ Connected!")

	// Listen for 5 seconds
	log.Println("👂 Receiving price updates (5 seconds)...")
	go func() {
		time.Sleep(5 * time.Second)
		close(stopC)
	}()

	<-doneC
	log.Printf("✅ Received %d price updates", eventCount)

	// Test depth stream
	log.Println("\n📡 Testing depth stream...")
	depthCount := 0
	depthHandler := func(event *futures.WsDepthEvent) {
		depthCount++
		if depthCount <= 3 {
			log.Printf("📊 Depth: %d bids, %d asks", len(event.Bids), len(event.Asks))
		}
	}

	doneC2, stopC2, err := futures.WsPartialDepthServeWithRate("BTCUSDT", 5, 100*time.Millisecond, depthHandler, errHandler)
	if err != nil {
		log.Printf("❌ Failed to connect depth: %v", err)
		return
	}

	go func() {
		time.Sleep(3 * time.Second)
		close(stopC2)
	}()

	<-doneC2
	log.Printf("✅ Received %d depth updates", depthCount)

	log.Println("\n✅ Futures Market Stream test completed")
}
