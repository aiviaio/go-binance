package main

import (
	"context"
	"log"
	"strings"

	"github.com/aiviaio/go-binance/v2/futures"
)

// TestFuturesMarketData tests Futures market data endpoints (trading schedule, RPI, ADL, insurance)
func TestFuturesMarketData(cfg *Config) {
	log.Println("\n" + strings.Repeat("=", 60))
	log.Println("TEST: Futures Market Data (Schedule, RPI, ADL, Insurance)")
	log.Println(strings.Repeat("=", 60))

	if cfg.FuturesAPIKey == "" || cfg.FuturesSecretKey == "" {
		log.Println("❌ SKIP: Futures API keys not configured")
		return
	}

	ctx := context.Background()
	client := futures.NewClient(cfg.FuturesAPIKey, cfg.FuturesSecretKey)

	// Test 1: Trading Schedule
	log.Println("\n📊 Testing GET /fapi/v1/tradingSchedule...")
	scheduleResp, err := client.NewTradingScheduleService().
		Symbol("BTCUSDT").
		Do(ctx)
	if err != nil {
		log.Printf("❌ Trading schedule: %v", err)
	} else {
		log.Printf("✅ Trading schedule: UpdateTime=%d, Markets=%d", scheduleResp.UpdateTime, len(scheduleResp.MarketSchedules))
		for market, schedule := range scheduleResp.MarketSchedules {
			log.Printf("   📅 %s: %d sessions", market, len(schedule.Sessions))
		}
	}

	// Test 2: RPI Depth (no limit parameter - uses default)
	log.Println("\n📊 Testing GET /fapi/v1/rpiDepth...")
	rpiDepth, err := client.NewRPIDepthService().
		Symbol("BTCUSDT").
		Do(ctx)
	if err != nil {
		log.Printf("❌ RPI Depth: %v", err)
	} else {
		log.Printf("✅ RPI Depth: LastUpdateID=%d, Bids=%d, Asks=%d",
			rpiDepth.LastUpdateID, len(rpiDepth.Bids), len(rpiDepth.Asks))
	}

	// Test 3: Symbol ADL Risk (requires open position to return data)
	log.Println("\n📊 Testing GET /fapi/v1/symbolAdlRisk...")
	adlRisks, err := client.NewSymbolADLRiskService().Do(ctx)
	if err != nil {
		log.Printf("❌ ADL Risk: %v", err)
	} else {
		log.Printf("✅ ADL Risk: %d positions with ADL data", len(adlRisks))
		for _, r := range adlRisks {
			log.Printf("   📊 %s: Quantile=%d, Side=%s", r.Symbol, r.ADLQuantile, r.PositionSide)
		}
	}

	// Test 4: Insurance Balance
	log.Println("\n📊 Testing GET /fapi/v1/insuranceBalance...")
	insuranceBalances, err := client.NewInsuranceBalanceService().Do(ctx)
	if err != nil {
		log.Printf("⚠️  Insurance Balance: %v", err)
	} else {
		log.Printf("✅ Insurance balances: %d assets", len(insuranceBalances))
		for i, b := range insuranceBalances {
			if i < 3 {
				log.Printf("   💰 %s: %s", b.Asset, b.Balance)
			}
		}
	}

	log.Println("\n✅ Futures Market Data test completed")
}
