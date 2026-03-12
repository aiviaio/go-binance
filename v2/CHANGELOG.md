# Changelog

## v2.1.49 (2026-03-12)

### Bug Fixes
- **Futures WebSocket API**: Fixed `WsRequestedUserDataStreamService` to use correct WebSocket API methods (`account.balance`, `account.position`) instead of invalid `REQUEST` method
- **Futures WebSocket API endpoint**: Fixed endpoint URL from `ws-api/v3` to `ws-fapi/v1`
- **Futures WebSocket architecture**: Now uses two separate connections:
  - User Data Stream (`fstream.binance.com/private/ws/<listenKey>`) for real-time events
  - WebSocket API (`ws-fapi.binance.com/ws-fapi/v1`) for request-response queries with signature authentication

### New Features
- **ALGO_UPDATE event parsing**: Added proper parsing for `ALGO_UPDATE` WebSocket event in User Data Stream
- **`UserDataEventTypeAlgoUpdate`**: New constant for algo order update events
- **Integration test**: Added `TestFuturesAlgoUpdateStream` for ALGO_UPDATE event verification

### Breaking Changes
- `NewWsRequestedUserDataStreamService()` now requires `secretKey` parameter for WebSocket API signature authentication

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
