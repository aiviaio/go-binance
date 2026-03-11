package main

import (
	"log"
	"os"

	"github.com/aiviaio/go-binance/v2"
	"github.com/aiviaio/go-binance/v2/futures"
)

type Config struct {
	SpotAPIKey        string
	SpotSecretKey     string
	FuturesAPIKey     string
	FuturesSecretKey  string
}

func LoadConfig() *Config {
	cfg := &Config{
		SpotAPIKey:       os.Getenv("BINANCE_SPOT_TESTNET_API_KEY"),
		SpotSecretKey:    os.Getenv("BINANCE_SPOT_TESTNET_SECRET_KEY"),
		FuturesAPIKey:    os.Getenv("BINANCE_FUTURES_TESTNET_API_KEY"),
		FuturesSecretKey: os.Getenv("BINANCE_FUTURES_TESTNET_SECRET_KEY"),
	}

	if cfg.SpotAPIKey == "" || cfg.SpotSecretKey == "" {
		log.Println("⚠️  Spot testnet keys not set")
	}
	if cfg.FuturesAPIKey == "" || cfg.FuturesSecretKey == "" {
		log.Println("⚠️  Futures testnet keys not set")
	}

	return cfg
}

func EnableTestnet() {
	binance.UseTestnet = true
	futures.UseTestnet = true
	log.Println("✅ Testnet mode enabled")
}
