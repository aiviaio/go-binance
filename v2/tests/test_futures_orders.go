package main

import (
	"context"
	"log"
	"strings"

	"github.com/aiviaio/go-binance/v2/futures"
)

// TestFuturesOrders tests Futures order operations
func TestFuturesOrders(cfg *Config) {
	log.Println("\n" + strings.Repeat("=", 60))
	log.Println("TEST: Futures Orders (Market, Limit)")
	log.Println(strings.Repeat("=", 60))

	if cfg.FuturesAPIKey == "" || cfg.FuturesSecretKey == "" {
		log.Println("❌ SKIP: Futures API keys not configured")
		return
	}

	ctx := context.Background()
	client := futures.NewClient(cfg.FuturesAPIKey, cfg.FuturesSecretKey)

	// Test 1: Get account info
	log.Println("\n📊 Getting futures account info...")
	account, err := client.NewGetAccountService().Do(ctx)
	if err != nil {
		log.Printf("❌ Failed to get account: %v", err)
		return
	}
	log.Printf("✅ Account loaded. Total Wallet Balance: %s USDT", account.TotalWalletBalance)

	// Show balances
	log.Println("💰 Balances:")
	for _, a := range account.Assets {
		if a.WalletBalance != "0" && a.WalletBalance != "0.00000000" {
			log.Printf("   %s: Wallet=%s", a.Asset, a.WalletBalance)
		}
	}

	// Test 2: Get positions
	log.Println("\n📊 Getting positions...")
	positions, err := client.NewGetPositionRiskService().Do(ctx)
	if err != nil {
		log.Printf("❌ Failed to get positions: %v", err)
	} else {
		hasPositions := false
		for _, p := range positions {
			if p.PositionAmt != "0" && p.PositionAmt != "0.000" {
				hasPositions = true
				log.Printf("   📈 %s: Amt=%s, Entry=%s, PNL=%s", p.Symbol, p.PositionAmt, p.EntryPrice, p.UnRealizedProfit)
			}
		}
		if !hasPositions {
			log.Println("   No open positions")
		}
	}

	// Test 3: Get open orders
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

	// Test 4: Place a LIMIT order BTCUSDT (notional > 100$)
	log.Println("\n📝 Placing LIMIT BUY order (BTCUSDT @ $50000, qty=0.002, notional ~$100)...")
	order, err := client.NewCreateOrderService().
		Symbol("BTCUSDT").
		Side(futures.SideTypeBuy).
		Type(futures.OrderTypeLimit).
		TimeInForce(futures.TimeInForceTypeGTC).
		Quantity("0.002").
		Price("50000").
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

	// Test 5: Place a MARKET order BTCUSDT (notional > 100$)
	log.Println("\n📝 Placing MARKET BUY order (BTCUSDT, 0.002 BTC, ~$170)...")
	marketOrder, err := client.NewCreateOrderService().
		Symbol("BTCUSDT").
		Side(futures.SideTypeBuy).
		Type(futures.OrderTypeMarket).
		Quantity("0.002").
		Do(ctx)
	if err != nil {
		log.Printf("❌ Market order failed: %v", err)
	} else {
		log.Printf("✅ Market order filled! OrderID: %d, Status: %s", marketOrder.OrderID, marketOrder.Status)

		// Close position
		log.Println("📝 Closing position with MARKET SELL...")
		closeOrder, err := client.NewCreateOrderService().
			Symbol("BTCUSDT").
			Side(futures.SideTypeSell).
			Type(futures.OrderTypeMarket).
			Quantity("0.002").
			Do(ctx)
		if err != nil {
			log.Printf("❌ Close order failed: %v", err)
		} else {
			log.Printf("✅ Position closed! OrderID: %d", closeOrder.OrderID)
		}
	}

	// Test 6: DOGUSDT - different precision and notional
	log.Println("\n📝 Testing DOGEUSDT (different precision)...")
	log.Println("📝 Placing LIMIT BUY order (DOGEUSDT @ $0.05, qty=2500, notional ~$125)...")
	dogOrder, err := client.NewCreateOrderService().
		Symbol("DOGEUSDT").
		Side(futures.SideTypeBuy).
		Type(futures.OrderTypeLimit).
		TimeInForce(futures.TimeInForceTypeGTC).
		Quantity("2500").
		Price("0.05").
		Do(ctx)
	if err != nil {
		log.Printf("❌ Failed to place DOGE order: %v", err)
	} else {
		log.Printf("✅ DOGE Order placed! OrderID: %d, Status: %s", dogOrder.OrderID, dogOrder.Status)

		// Cancel
		log.Println("🗑️  Canceling DOGE order...")
		_, err = client.NewCancelOrderService().
			Symbol("DOGEUSDT").
			OrderID(dogOrder.OrderID).
			Do(ctx)
		if err != nil {
			log.Printf("❌ Failed to cancel: %v", err)
		} else {
			log.Println("✅ DOGE Order canceled")
		}
	}

	// Test 7: DOGUSDT MARKET order
	log.Println("\n📝 Placing MARKET BUY order (DOGEUSDT, 500 DOGE)...")
	dogMarket, err := client.NewCreateOrderService().
		Symbol("DOGEUSDT").
		Side(futures.SideTypeBuy).
		Type(futures.OrderTypeMarket).
		Quantity("500").
		Do(ctx)
	if err != nil {
		log.Printf("❌ DOGE Market order failed: %v", err)
	} else {
		log.Printf("✅ DOGE Market order filled! OrderID: %d, Status: %s, AvgPrice: %s", dogMarket.OrderID, dogMarket.Status, dogMarket.AvgPrice)

		// Close position
		log.Println("📝 Closing DOGE position...")
		dogClose, err := client.NewCreateOrderService().
			Symbol("DOGEUSDT").
			Side(futures.SideTypeSell).
			Type(futures.OrderTypeMarket).
			Quantity("500").
			Do(ctx)
		if err != nil {
			log.Printf("❌ DOGE Close failed: %v", err)
		} else {
			log.Printf("✅ DOGE Position closed! OrderID: %d", dogClose.OrderID)
		}
	}

	// Test 8: SOLUSDT - another asset
	log.Println("\n📝 Testing SOLUSDT...")
	log.Println("📝 Placing MARKET BUY order (SOLUSDT, 1 SOL)...")
	solMarket, err := client.NewCreateOrderService().
		Symbol("SOLUSDT").
		Side(futures.SideTypeBuy).
		Type(futures.OrderTypeMarket).
		Quantity("1").
		Do(ctx)
	if err != nil {
		log.Printf("❌ SOL Market order failed: %v", err)
	} else {
		log.Printf("✅ SOL Market order filled! OrderID: %d, Status: %s, AvgPrice: %s", solMarket.OrderID, solMarket.Status, solMarket.AvgPrice)

		// Close position
		log.Println("📝 Closing SOL position...")
		solClose, err := client.NewCreateOrderService().
			Symbol("SOLUSDT").
			Side(futures.SideTypeSell).
			Type(futures.OrderTypeMarket).
			Quantity("1").
			Do(ctx)
		if err != nil {
			log.Printf("❌ SOL Close failed: %v", err)
		} else {
			log.Printf("✅ SOL Position closed! OrderID: %d", solClose.OrderID)
		}
	}

	log.Println("\n✅ Futures Orders test completed")
}

// TestFuturesBalance tests getting Futures balances and positions
func TestFuturesBalance(cfg *Config) {
	log.Println("\n" + strings.Repeat("=", 60))
	log.Println("TEST: Futures Balance & Positions")
	log.Println(strings.Repeat("=", 60))

	if cfg.FuturesAPIKey == "" || cfg.FuturesSecretKey == "" {
		log.Println("❌ SKIP: Futures API keys not configured")
		return
	}

	ctx := context.Background()
	client := futures.NewClient(cfg.FuturesAPIKey, cfg.FuturesSecretKey)

	// Get balance
	balances, err := client.NewGetBalanceService().Do(ctx)
	if err != nil {
		log.Printf("❌ Failed to get balance: %v", err)
		return
	}

	log.Println("✅ Balances:")
	for _, b := range balances {
		if b.Balance != "0" && b.Balance != "0.00000000" {
			log.Printf("   💰 %s: Balance=%s, Available=%s", b.Asset, b.Balance, b.AvailableBalance)
		}
	}
}
