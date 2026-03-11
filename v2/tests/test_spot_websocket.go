package main

import (
	"log"
	"strings"
	"time"

	"github.com/aiviaio/go-binance/v2"
)

// TestSpotWebSocketAPI tests the new Spot WebSocket API for user data stream
func TestSpotWebSocketAPI(cfg *Config) {
	log.Println("\n" + strings.Repeat("=", 60))
	log.Println("TEST: Spot WebSocket API User Data Stream")
	log.Println(strings.Repeat("=", 60))

	if cfg.SpotAPIKey == "" || cfg.SpotSecretKey == "" {
		log.Println("❌ SKIP: Spot API keys not configured")
		return
	}

	errHandler := func(err error) {
		log.Printf("❌ [Spot WS API] Error: %v", err)
	}

	// Create WebSocket API client
	log.Println("📡 Creating WebSocket API client...")
	wsClient, err := binance.NewWsAPIClient(cfg.SpotAPIKey, cfg.SpotSecretKey, errHandler)
	if err != nil {
		log.Printf("❌ Failed to create client: %v", err)
		return
	}
	log.Println("✅ WebSocket API client created")

	// Subscribe to user data stream
	log.Println("📡 Subscribing to user data stream...")
	subscriptionID, err := wsClient.SubscribeUserDataStream()
	if err != nil {
		log.Printf("❌ Failed to subscribe: %v", err)
		wsClient.Close()
		return
	}
	log.Printf("✅ Subscribed! SubscriptionID: %d", subscriptionID)

	// Listen for events for 10 seconds
	log.Println("👂 Listening for events (10 seconds)...")
	go func() {
		userDataChan := wsClient.UserDataChan()
		for event := range userDataChan {
			log.Printf("📨 Event: %s", event.Event)
			if event.BalanceUpdate.Asset != "" {
				log.Printf("   💰 Balance Update: %s %s", event.BalanceUpdate.Asset, event.BalanceUpdate.Change)
			}
			if event.OrderUpdate.Symbol != "" {
				log.Printf("   📋 Order Update: %s %s %s", event.OrderUpdate.Symbol, event.OrderUpdate.Side, event.OrderUpdate.Status)
			}
		}
	}()

	time.Sleep(10 * time.Second)

	// Unsubscribe
	log.Println("📡 Unsubscribing...")
	err = wsClient.UnsubscribeUserDataStream(subscriptionID)
	if err != nil {
		log.Printf("⚠️  Unsubscribe warning: %v", err)
	}

	wsClient.Close()
	log.Println("✅ Spot WebSocket API test completed")
}
