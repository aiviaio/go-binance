package main

import (
	"context"
	"log"
	"strings"

	"github.com/aiviaio/go-binance/v2"
)

// TestSpotOrders tests Spot order operations
func TestSpotOrders(cfg *Config) {
	log.Println("\n" + strings.Repeat("=", 60))
	log.Println("TEST: Spot Orders (Market, Limit)")
	log.Println(strings.Repeat("=", 60))

	if cfg.SpotAPIKey == "" || cfg.SpotSecretKey == "" {
		log.Println("❌ SKIP: Spot API keys not configured")
		return
	}

	ctx := context.Background()
	client := binance.NewClient(cfg.SpotAPIKey, cfg.SpotSecretKey)

	// Test 1: Get account info
	log.Println("\n📊 Getting account info...")
	account, err := client.NewGetAccountService().Do(ctx)
	if err != nil {
		log.Printf("❌ Failed to get account: %v", err)
		return
	}
	log.Printf("✅ Account loaded. Balances with value:")
	for _, b := range account.Balances {
		if b.Free != "0" && b.Free != "0.00000000" {
			log.Printf("   💰 %s: Free=%s, Locked=%s", b.Asset, b.Free, b.Locked)
		}
	}

	// Test 2: Get open orders
	log.Println("\n📋 Getting open orders...")
	openOrders, err := client.NewListOpenOrdersService().Do(ctx)
	if err != nil {
		log.Printf("❌ Failed to get open orders: %v", err)
	} else {
		log.Printf("✅ Open orders: %d", len(openOrders))
		for _, o := range openOrders {
			log.Printf("   📝 %s %s %s @ %s (qty: %s)", o.Symbol, o.Side, o.Type, o.Price, o.OrigQuantity)
		}
	}

	// Test 3: Place a LIMIT order (very low price so it won't fill)
	log.Println("\n📝 Placing LIMIT BUY order (BTCUSDT @ $1000 - won't fill)...")
	order, err := client.NewCreateOrderService().
		Symbol("BTCUSDT").
		Side(binance.SideTypeBuy).
		Type(binance.OrderTypeLimit).
		TimeInForce(binance.TimeInForceTypeGTC).
		Quantity("0.001").
		Price("1000").
		Do(ctx)
	if err != nil {
		log.Printf("❌ Failed to place order: %v", err)
	} else {
		log.Printf("✅ Order placed! OrderID: %d, Status: %s", order.OrderID, order.Status)

		// Cancel the order
		log.Println("🗑️  Canceling order...")
		_, err = client.NewCancelOrderService().
			Symbol("BTCUSDT").
			OrderID(order.OrderID).
			Do(ctx)
		if err != nil {
			log.Printf("❌ Failed to cancel: %v", err)
		} else {
			log.Println("✅ Order canceled")
		}
	}

	// Test 4: Place a MARKET order (small amount)
	log.Println("\n📝 Placing MARKET BUY order (BTCUSDT, 0.0001 BTC)...")
	marketOrder, err := client.NewCreateOrderService().
		Symbol("BTCUSDT").
		Side(binance.SideTypeBuy).
		Type(binance.OrderTypeMarket).
		Quantity("0.0001").
		Do(ctx)
	if err != nil {
		log.Printf("⚠️  Market order failed (may be min qty issue on testnet): %v", err)
	} else {
		log.Printf("✅ Market order filled! OrderID: %d, Status: %s", marketOrder.OrderID, marketOrder.Status)
	}

	log.Println("\n✅ Spot Orders test completed")
}

// TestSpotBalance tests getting Spot balances
func TestSpotBalance(cfg *Config) {
	log.Println("\n" + strings.Repeat("=", 60))
	log.Println("TEST: Spot Balance")
	log.Println(strings.Repeat("=", 60))

	if cfg.SpotAPIKey == "" || cfg.SpotSecretKey == "" {
		log.Println("❌ SKIP: Spot API keys not configured")
		return
	}

	ctx := context.Background()
	client := binance.NewClient(cfg.SpotAPIKey, cfg.SpotSecretKey)

	account, err := client.NewGetAccountService().Do(ctx)
	if err != nil {
		log.Printf("❌ Failed to get account: %v", err)
		return
	}

	log.Println("✅ Balances:")
	for _, b := range account.Balances {
		if b.Free != "0" && b.Free != "0.00000000" {
			log.Printf("   💰 %s: Free=%s, Locked=%s", b.Asset, b.Free, b.Locked)
		}
	}
}
