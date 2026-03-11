# Changelog

## v2.1.48 (2026-03-11)

### Breaking Changes
- Conditional order types (`STOP_MARKET`, `TAKE_PROFIT_MARKET`, `TRAILING_STOP_MARKET`) are now blocked on standard Futures order endpoints. Use the new Algo Order API instead.

### New Features

#### Spot
- **WebSocket API for User Data Stream**: New `WsAPIClient` with `SubscribeUserDataStream()` method replacing deprecated REST-based listenKey management
- Updated `/api/v1/trades` to `/api/v3/trades` (v1 deprecated, retiring 2026-03-25)

#### Futures
- **New WebSocket endpoints**: 
  - Market data: `wss://fstream.binance.com/market/ws`
  - Private/User data: `wss://fstream.binance.com/private/ws`
- **Algo Order API** (`/fapi/v1/algoOrder`):
  - `NewCreateAlgoOrderService()` - Create algo orders (STOP_MARKET, TAKE_PROFIT, TRAILING_STOP_MARKET, etc.)
  - `NewCancelAlgoOrderService()` - Cancel algo order by ID
  - `NewCancelAllAlgoOrdersService()` - Cancel all open algo orders
  - `NewGetAlgoOrderService()` - Query single algo order
  - `NewListOpenAlgoOrdersService()` - List open algo orders
  - `NewListAllAlgoOrdersService()` - List all algo orders (history)
- **ALGO_UPDATE event**: New `WsAlgoUpdate` struct in User Data Stream for algo order updates
- **Trading Schedule** (`/fapi/v1/tradingSchedule`): Get trading session schedules for TradFi-Perps
- **RPI Depth** (`/fapi/v1/rpiDepth`): Get RPI order book
- **Symbol ADL Risk** (`/fapi/v1/symbolAdlRisk`): Get ADL risk rating for positions
- **Insurance Balance** (`/fapi/v1/insuranceBalance`): Get insurance fund balance

### Deprecation Warnings
- Added deprecation warning for `WsAllMarketTickerServe` (`!ticker@arr` stream retiring 2026-03-26)
- Added deprecation warning for REST-based listenKey endpoints (use WebSocket API instead)

### Bug Fixes
- Fixed `secType` for Futures listenKey endpoints (changed from `secTypeSigned` to `secTypeAPIKey`)
