package futures

import (
	"context"
	"encoding/json"
	"net/http"
)

// TradingScheduleService get trading session schedules
type TradingScheduleService struct {
	c      *Client
	symbol *string
}

// Symbol set symbol
func (s *TradingScheduleService) Symbol(symbol string) *TradingScheduleService {
	s.symbol = &symbol
	return s
}

// Do send request
func (s *TradingScheduleService) Do(ctx context.Context, opts ...RequestOption) (*TradingScheduleResponse, error) {
	r := &request{
		method:   http.MethodGet,
		endpoint: "/fapi/v1/tradingSchedule",
		secType:  secTypeNone,
	}
	if s.symbol != nil {
		r.setParam("symbol", *s.symbol)
	}

	data, _, err := s.c.callAPI(ctx, r, opts...)
	if err != nil {
		return nil, err
	}
	res := new(TradingScheduleResponse)
	err = json.Unmarshal(data, res)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// TradingScheduleResponse define trading schedule response
type TradingScheduleResponse struct {
	UpdateTime      int64                       `json:"updateTime"`
	MarketSchedules map[string]*MarketSchedule  `json:"marketSchedules"`
}

// MarketSchedule define market schedule
type MarketSchedule struct {
	Sessions []*TradingSession `json:"sessions"`
}

// TradingSession define trading session
type TradingSession struct {
	StartTime int64  `json:"startTime"`
	EndTime   int64  `json:"endTime"`
	Type      string `json:"type"`
}

// RPIDepthService get RPI order book
type RPIDepthService struct {
	c      *Client
	symbol string
	limit  *int
}

// Symbol set symbol
func (s *RPIDepthService) Symbol(symbol string) *RPIDepthService {
	s.symbol = symbol
	return s
}

// Limit set limit
func (s *RPIDepthService) Limit(limit int) *RPIDepthService {
	s.limit = &limit
	return s
}

// Do send request
func (s *RPIDepthService) Do(ctx context.Context, opts ...RequestOption) (*RPIDepth, error) {
	r := &request{
		method:   http.MethodGet,
		endpoint: "/fapi/v1/rpiDepth",
		secType:  secTypeNone,
	}
	r.setParam("symbol", s.symbol)
	if s.limit != nil {
		r.setParam("limit", *s.limit)
	}

	data, _, err := s.c.callAPI(ctx, r, opts...)
	if err != nil {
		return nil, err
	}
	res := new(RPIDepth)
	err = json.Unmarshal(data, res)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// RPIDepth define RPI order book depth
type RPIDepth struct {
	LastUpdateID int64      `json:"lastUpdateId"`
	Bids         [][]string `json:"bids"`
	Asks         [][]string `json:"asks"`
}

// SymbolADLRiskService get ADL risk rating for symbol
type SymbolADLRiskService struct {
	c      *Client
	symbol *string
}

// Symbol set symbol (optional - if not set, returns all positions with ADL risk)
func (s *SymbolADLRiskService) Symbol(symbol string) *SymbolADLRiskService {
	s.symbol = &symbol
	return s
}

// Do send request
func (s *SymbolADLRiskService) Do(ctx context.Context, opts ...RequestOption) ([]*SymbolADLRisk, error) {
	r := &request{
		method:   http.MethodGet,
		endpoint: "/fapi/v1/symbolAdlRisk",
		secType:  secTypeSigned,
	}
	if s.symbol != nil {
		r.setParam("symbol", *s.symbol)
	}

	data, _, err := s.c.callAPI(ctx, r, opts...)
	if err != nil {
		return nil, err
	}
	var res []*SymbolADLRisk
	err = json.Unmarshal(data, &res)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// SymbolADLRisk define ADL risk rating
type SymbolADLRisk struct {
	Symbol       string `json:"symbol"`
	ADLQuantile  int    `json:"adlQuantile"`
	PositionSide string `json:"positionSide"`
}

// InsuranceBalanceService get insurance fund balance
type InsuranceBalanceService struct {
	c      *Client
	symbol *string
}

// Symbol set symbol
func (s *InsuranceBalanceService) Symbol(symbol string) *InsuranceBalanceService {
	s.symbol = &symbol
	return s
}

// Do send request
func (s *InsuranceBalanceService) Do(ctx context.Context, opts ...RequestOption) ([]*InsuranceBalance, error) {
	r := &request{
		method:   http.MethodGet,
		endpoint: "/fapi/v1/insuranceBalance",
		secType:  secTypeNone,
	}
	if s.symbol != nil {
		r.setParam("symbol", *s.symbol)
	}

	data, _, err := s.c.callAPI(ctx, r, opts...)
	if err != nil {
		return nil, err
	}
	var res []*InsuranceBalance
	err = json.Unmarshal(data, &res)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// InsuranceBalance define insurance fund balance
type InsuranceBalance struct {
	Asset   string `json:"asset"`
	Balance string `json:"balance"`
	Time    int64  `json:"time"`
}
