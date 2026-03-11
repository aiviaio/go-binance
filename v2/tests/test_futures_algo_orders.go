package main

import (
	"context"
	"log"
	"strings"

	"github.com/aiviaio/go-binance/v2/futures"
)

// TestFuturesAlgoOrders tests the NEW Algo Order API (POST /fapi/v1/algoOrder)
func TestFuturesAlgoOrders(cfg *Config) {
	log.Println("\n" + strings.Repeat("=", 60))
	log.Println("TEST: Futures Algo Order API (NEW /fapi/v1/algoOrder)")
	log.Println(strings.Repeat("=", 60))

	if cfg.FuturesAPIKey == "" || cfg.FuturesSecretKey == "" {
		log.Println("❌ SKIP: Futures API keys not configured")
		return
	}

	ctx := context.Background()
	client := futures.NewClient(cfg.FuturesAPIKey, cfg.FuturesSecretKey)

	// Test 1: Create TAKE_PROFIT algo order
	log.Println("\n📝 Creating TAKE_PROFIT algo order via NEW API...")
	algoOrder, err := client.NewCreateAlgoOrderService().
		Symbol("BTCUSDT").
		Side(futures.SideTypeSell).
		Type(futures.OrderTypeTakeProfit).
		Quantity("0.002").
		TriggerPrice("150000").
		Price("150000").
		Do(ctx)
	if err != nil {
		log.Printf("❌ Failed to create algo order: %v", err)
	} else {
		log.Printf("✅ Algo order created! AlgoID: %d, Status: %s", algoOrder.AlgoID, algoOrder.AlgoStatus)
		log.Printf("   Symbol: %s, Side: %s, Type: %s", algoOrder.Symbol, algoOrder.Side, algoOrder.OrderType)
		log.Printf("   TriggerPrice: %s, Quantity: %s", algoOrder.TriggerPrice, algoOrder.Quantity)

		// Cancel the algo order
		log.Println("\n🗑️  Canceling algo order...")
		cancelRes, err := client.NewCancelAlgoOrderService().
			Symbol("BTCUSDT").
			AlgoID(algoOrder.AlgoID).
			Do(ctx)
		if err != nil {
			log.Printf("❌ Failed to cancel: %v", err)
		} else {
			log.Printf("✅ Algo order canceled! AlgoID: %d, Status: %s", cancelRes.AlgoID, cancelRes.AlgoStatus)
		}
	}

	// Test 2: Create STOP_MARKET algo order
	// BUY STOP_MARKET triggers when price >= triggerPrice, so triggerPrice must be ABOVE current price
	log.Println("\n📝 Creating STOP_MARKET algo order via NEW API...")
	stopOrder, err := client.NewCreateAlgoOrderService().
		Symbol("BTCUSDT").
		Side(futures.SideTypeBuy).
		Type(futures.OrderTypeStopMarket).
		Quantity("0.002").
		TriggerPrice("100000").
		Do(ctx)
	if err != nil {
		log.Printf("❌ Failed to create STOP_MARKET: %v", err)
	} else {
		log.Printf("✅ STOP_MARKET created! AlgoID: %d, Status: %s", stopOrder.AlgoID, stopOrder.AlgoStatus)

		// Cancel
		log.Println("�️  Canceling...")
		_, err = client.NewCancelAlgoOrderService().
			Symbol("BTCUSDT").
			AlgoID(stopOrder.AlgoID).
			Do(ctx)
		if err != nil {
			log.Printf("❌ Failed to cancel: %v", err)
		} else {
			log.Println("✅ Canceled")
		}
	}

	// Test 3: List open algo orders
	log.Println("\n📋 Listing open algo orders...")
	openOrders, err := client.NewListOpenAlgoOrdersService().Do(ctx)
	if err != nil {
		log.Printf("❌ Failed to list open algo orders: %v", err)
	} else {
		log.Printf("✅ Open algo orders: %d", len(openOrders))
		for _, o := range openOrders {
			log.Printf("   📝 AlgoID=%d, %s %s %s @ %s", o.AlgoID, o.Symbol, o.Side, o.OrderType, o.TriggerPrice)
		}
	}

	// Test 4: List all algo orders (history)
	log.Println("\n📋 Listing all algo orders (history)...")
	allOrders, err := client.NewListAllAlgoOrdersService().
		Symbol("BTCUSDT").
		Limit(5).
		Do(ctx)
	if err != nil {
		log.Printf("❌ Failed to list all algo orders: %v", err)
	} else {
		log.Printf("✅ All algo orders: %d", len(allOrders))
		for _, o := range allOrders {
			log.Printf("   📝 AlgoID=%d, %s %s %s, Status=%s", o.AlgoID, o.Symbol, o.Side, o.OrderType, o.AlgoStatus)
		}
	}

	log.Println("\n✅ Futures Algo Order API test completed")
}
