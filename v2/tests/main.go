package main

import (
	"flag"
	"log"
	"strings"
)

func main() {
	// Parse command line flags
	testName := flag.String("test", "all", "Test to run: all, spot-ws, spot-orders, spot-balance, futures-ws, futures-market, futures-orders, futures-balance, futures-algo")
	flag.Parse()

	// Load config and enable testnet
	cfg := LoadConfig()
	EnableTestnet()

	log.Println("\n" + strings.Repeat("=", 60))
	log.Println("BINANCE GO LIBRARY - COMPREHENSIVE TEST SUITE")
	log.Println(strings.Repeat("=", 60))

	switch *testName {
	case "all":
		runAllTests(cfg)
	case "spot-ws":
		TestSpotWebSocketAPI(cfg)
	case "spot-orders":
		TestSpotOrders(cfg)
	case "spot-balance":
		TestSpotBalance(cfg)
	case "spot-trades":
		TestSpotRecentTrades(cfg)
	case "futures-ws":
		TestFuturesUserDataStream(cfg)
	case "futures-market":
		TestFuturesMarketStream(cfg)
	case "futures-orders":
		TestFuturesOrders(cfg)
	case "futures-balance":
		TestFuturesBalance(cfg)
	case "futures-algo":
		TestFuturesAlgoOrders(cfg)
	case "futures-trades":
		TestFuturesHistoricalTrades(cfg)
	case "futures-info":
		TestFuturesExchangeInfo(cfg)
	case "futures-market-data":
		TestFuturesMarketData(cfg)
	default:
		log.Printf("Unknown test: %s", *testName)
		log.Println("Available tests: all, spot-ws, spot-orders, spot-balance, futures-ws, futures-market, futures-orders, futures-balance, futures-algo")
	}

	log.Println("\n" + strings.Repeat("=", 60))
	log.Println("TEST SUITE COMPLETED")
	log.Println(strings.Repeat("=", 60))
}

func runAllTests(cfg *Config) {
	// Spot tests
	TestSpotBalance(cfg)
	TestSpotRecentTrades(cfg)
	TestSpotWebSocketAPI(cfg)
	TestSpotOrders(cfg)

	// Futures tests
	TestFuturesBalance(cfg)
	TestFuturesExchangeInfo(cfg)
	TestFuturesHistoricalTrades(cfg)
	TestFuturesMarketStream(cfg)
	TestFuturesUserDataStream(cfg)
	TestFuturesOrders(cfg)
	TestFuturesAlgoOrders(cfg)
	TestFuturesMarketData(cfg)
}
