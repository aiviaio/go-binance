package futures

import (
	"context"
	"encoding/json"
	"net/http"
)

// AlgoOrderType define algo order type
type AlgoOrderType string

// AlgoOrderStatus define algo order status
type AlgoOrderStatus string

// AlgoType define algo type
type AlgoType string

const (
	AlgoTypeConditional AlgoType = "CONDITIONAL"
)

const (
	AlgoOrderStatusNew       AlgoOrderStatus = "NEW"
	AlgoOrderStatusTriggered AlgoOrderStatus = "TRIGGERED"
	AlgoOrderStatusCancelled AlgoOrderStatus = "CANCELLED"
	AlgoOrderStatusFailed    AlgoOrderStatus = "FAILED"
	AlgoOrderStatusExpired   AlgoOrderStatus = "EXPIRED"
)

// CreateAlgoOrderService create algo order
type CreateAlgoOrderService struct {
	c                       *Client
	symbol                  string
	side                    SideType
	positionSide            *PositionSideType
	quantity                *string
	algoType                AlgoType
	orderType               OrderType
	timeInForce             *TimeInForceType
	price                   *string
	triggerPrice            *string
	activatePrice           *string
	callbackRate            *string
	workingType             *WorkingType
	priceProtect            *bool
	reduceOnly              *bool
	closePosition           *bool
	priceMatch              *string
	selfTradePreventionMode *string
	newClientOrderID        *string
	goodTillDate            *int64
}

// Symbol set symbol
func (s *CreateAlgoOrderService) Symbol(symbol string) *CreateAlgoOrderService {
	s.symbol = symbol
	return s
}

// Side set side
func (s *CreateAlgoOrderService) Side(side SideType) *CreateAlgoOrderService {
	s.side = side
	return s
}

// PositionSide set positionSide
func (s *CreateAlgoOrderService) PositionSide(positionSide PositionSideType) *CreateAlgoOrderService {
	s.positionSide = &positionSide
	return s
}

// Quantity set quantity
func (s *CreateAlgoOrderService) Quantity(quantity string) *CreateAlgoOrderService {
	s.quantity = &quantity
	return s
}

// Type set orderType
func (s *CreateAlgoOrderService) Type(orderType OrderType) *CreateAlgoOrderService {
	s.orderType = orderType
	return s
}

// TimeInForce set timeInForce
func (s *CreateAlgoOrderService) TimeInForce(timeInForce TimeInForceType) *CreateAlgoOrderService {
	s.timeInForce = &timeInForce
	return s
}

// Price set price
func (s *CreateAlgoOrderService) Price(price string) *CreateAlgoOrderService {
	s.price = &price
	return s
}

// TriggerPrice set triggerPrice
func (s *CreateAlgoOrderService) TriggerPrice(triggerPrice string) *CreateAlgoOrderService {
	s.triggerPrice = &triggerPrice
	return s
}

// ActivatePrice set activatePrice (for TRAILING_STOP_MARKET)
func (s *CreateAlgoOrderService) ActivatePrice(activatePrice string) *CreateAlgoOrderService {
	s.activatePrice = &activatePrice
	return s
}

// CallbackRate set callbackRate (for TRAILING_STOP_MARKET)
func (s *CreateAlgoOrderService) CallbackRate(callbackRate string) *CreateAlgoOrderService {
	s.callbackRate = &callbackRate
	return s
}

// WorkingType set workingType
func (s *CreateAlgoOrderService) WorkingType(workingType WorkingType) *CreateAlgoOrderService {
	s.workingType = &workingType
	return s
}

// PriceProtect set priceProtect
func (s *CreateAlgoOrderService) PriceProtect(priceProtect bool) *CreateAlgoOrderService {
	s.priceProtect = &priceProtect
	return s
}

// ReduceOnly set reduceOnly
func (s *CreateAlgoOrderService) ReduceOnly(reduceOnly bool) *CreateAlgoOrderService {
	s.reduceOnly = &reduceOnly
	return s
}

// ClosePosition set closePosition
func (s *CreateAlgoOrderService) ClosePosition(closePosition bool) *CreateAlgoOrderService {
	s.closePosition = &closePosition
	return s
}

// PriceMatch set priceMatch
func (s *CreateAlgoOrderService) PriceMatch(priceMatch string) *CreateAlgoOrderService {
	s.priceMatch = &priceMatch
	return s
}

// SelfTradePreventionMode set selfTradePreventionMode
func (s *CreateAlgoOrderService) SelfTradePreventionMode(mode string) *CreateAlgoOrderService {
	s.selfTradePreventionMode = &mode
	return s
}

// NewClientOrderID set newClientOrderId
func (s *CreateAlgoOrderService) NewClientOrderID(newClientOrderID string) *CreateAlgoOrderService {
	s.newClientOrderID = &newClientOrderID
	return s
}

// GoodTillDate set goodTillDate
func (s *CreateAlgoOrderService) GoodTillDate(goodTillDate int64) *CreateAlgoOrderService {
	s.goodTillDate = &goodTillDate
	return s
}

// AlgoType set algoType (default CONDITIONAL)
func (s *CreateAlgoOrderService) AlgoType(algoType AlgoType) *CreateAlgoOrderService {
	s.algoType = algoType
	return s
}

// Do send request
func (s *CreateAlgoOrderService) Do(ctx context.Context, opts ...RequestOption) (*AlgoOrder, error) {
	r := &request{
		method:   http.MethodPost,
		endpoint: "/fapi/v1/algoOrder",
		secType:  secTypeSigned,
	}
	r.setFormParam("symbol", s.symbol)
	r.setFormParam("side", s.side)
	r.setFormParam("type", s.orderType)
	if s.algoType != "" {
		r.setFormParam("algoType", s.algoType)
	} else {
		r.setFormParam("algoType", AlgoTypeConditional)
	}

	if s.positionSide != nil {
		r.setFormParam("positionSide", *s.positionSide)
	}
	if s.quantity != nil {
		r.setFormParam("quantity", *s.quantity)
	}
	if s.timeInForce != nil {
		r.setFormParam("timeInForce", *s.timeInForce)
	}
	if s.price != nil {
		r.setFormParam("price", *s.price)
	}
	if s.triggerPrice != nil {
		r.setFormParam("triggerPrice", *s.triggerPrice)
	}
	if s.activatePrice != nil {
		r.setFormParam("activatePrice", *s.activatePrice)
	}
	if s.callbackRate != nil {
		r.setFormParam("callbackRate", *s.callbackRate)
	}
	if s.workingType != nil {
		r.setFormParam("workingType", *s.workingType)
	}
	if s.priceProtect != nil {
		r.setFormParam("priceProtect", *s.priceProtect)
	}
	if s.reduceOnly != nil {
		r.setFormParam("reduceOnly", *s.reduceOnly)
	}
	if s.closePosition != nil {
		r.setFormParam("closePosition", *s.closePosition)
	}
	if s.priceMatch != nil {
		r.setFormParam("priceMatch", *s.priceMatch)
	}
	if s.selfTradePreventionMode != nil {
		r.setFormParam("selfTradePreventionMode", *s.selfTradePreventionMode)
	}
	if s.newClientOrderID != nil {
		r.setFormParam("newClientOrderId", *s.newClientOrderID)
	}
	if s.goodTillDate != nil {
		r.setFormParam("goodTillDate", *s.goodTillDate)
	}

	data, _, err := s.c.callAPI(ctx, r, opts...)
	if err != nil {
		return nil, err
	}
	res := new(AlgoOrder)
	err = json.Unmarshal(data, res)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// AlgoOrder define algo order info
type AlgoOrder struct {
	AlgoID                  int64           `json:"algoId"`
	ClientAlgoID            string          `json:"clientAlgoId"`
	AlgoType                AlgoType        `json:"algoType"`
	OrderType               OrderType       `json:"orderType"`
	Symbol                  string          `json:"symbol"`
	Side                    SideType        `json:"side"`
	PositionSide            PositionSideType `json:"positionSide"`
	TimeInForce             TimeInForceType `json:"timeInForce"`
	Quantity                string          `json:"quantity"`
	AlgoStatus              AlgoOrderStatus `json:"algoStatus"`
	TriggerPrice            string          `json:"triggerPrice"`
	Price                   string          `json:"price"`
	IcebergQuantity         string          `json:"icebergQuantity"`
	SelfTradePreventionMode string          `json:"selfTradePreventionMode"`
	WorkingType             WorkingType     `json:"workingType"`
	PriceMatch              string          `json:"priceMatch"`
	ClosePosition           bool            `json:"closePosition"`
	PriceProtect            bool            `json:"priceProtect"`
	ReduceOnly              bool            `json:"reduceOnly"`
	ActivatePrice           string          `json:"activatePrice"`
	CallbackRate            string          `json:"callbackRate"`
	CreateTime              int64           `json:"createTime"`
	UpdateTime              int64           `json:"updateTime"`
	TriggerTime             int64           `json:"triggerTime"`
	GoodTillDate            int64           `json:"goodTillDate"`
}

// CancelAlgoOrderService cancel algo order
type CancelAlgoOrderService struct {
	c      *Client
	symbol string
	algoID int64
}

// Symbol set symbol
func (s *CancelAlgoOrderService) Symbol(symbol string) *CancelAlgoOrderService {
	s.symbol = symbol
	return s
}

// AlgoID set algoId
func (s *CancelAlgoOrderService) AlgoID(algoID int64) *CancelAlgoOrderService {
	s.algoID = algoID
	return s
}

// Do send request
func (s *CancelAlgoOrderService) Do(ctx context.Context, opts ...RequestOption) (*CancelAlgoOrderResponse, error) {
	r := &request{
		method:   http.MethodDelete,
		endpoint: "/fapi/v1/algoOrder",
		secType:  secTypeSigned,
	}
	r.setFormParam("symbol", s.symbol)
	r.setFormParam("algoId", s.algoID)

	data, _, err := s.c.callAPI(ctx, r, opts...)
	if err != nil {
		return nil, err
	}
	res := new(CancelAlgoOrderResponse)
	err = json.Unmarshal(data, res)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// CancelAlgoOrderResponse define cancel algo order response
type CancelAlgoOrderResponse struct {
	AlgoID       int64           `json:"algoId"`
	AlgoStatus   AlgoOrderStatus `json:"algoStatus"`
	ClientAlgoID string          `json:"clientAlgoId"`
	Code         string          `json:"code"`
	Msg          string          `json:"msg"`
}

// CancelAllAlgoOrdersService cancel all open algo orders
type CancelAllAlgoOrdersService struct {
	c      *Client
	symbol string
}

// Symbol set symbol
func (s *CancelAllAlgoOrdersService) Symbol(symbol string) *CancelAllAlgoOrdersService {
	s.symbol = symbol
	return s
}

// Do send request
func (s *CancelAllAlgoOrdersService) Do(ctx context.Context, opts ...RequestOption) (*CancelAllAlgoOrdersResponse, error) {
	r := &request{
		method:   http.MethodDelete,
		endpoint: "/fapi/v1/algoOpenOrders",
		secType:  secTypeSigned,
	}
	r.setFormParam("symbol", s.symbol)

	data, _, err := s.c.callAPI(ctx, r, opts...)
	if err != nil {
		return nil, err
	}
	res := new(CancelAllAlgoOrdersResponse)
	err = json.Unmarshal(data, res)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// CancelAllAlgoOrdersResponse define cancel all algo orders response
type CancelAllAlgoOrdersResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

// GetAlgoOrderService get algo order
type GetAlgoOrderService struct {
	c      *Client
	symbol string
	algoID *int64
}

// Symbol set symbol
func (s *GetAlgoOrderService) Symbol(symbol string) *GetAlgoOrderService {
	s.symbol = symbol
	return s
}

// AlgoID set algoId
func (s *GetAlgoOrderService) AlgoID(algoID int64) *GetAlgoOrderService {
	s.algoID = &algoID
	return s
}

// Do send request
func (s *GetAlgoOrderService) Do(ctx context.Context, opts ...RequestOption) (*AlgoOrder, error) {
	r := &request{
		method:   http.MethodGet,
		endpoint: "/fapi/v1/algoOrder",
		secType:  secTypeSigned,
	}
	r.setParam("symbol", s.symbol)
	if s.algoID != nil {
		r.setParam("algoId", *s.algoID)
	}

	data, _, err := s.c.callAPI(ctx, r, opts...)
	if err != nil {
		return nil, err
	}
	res := new(AlgoOrder)
	err = json.Unmarshal(data, res)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// ListOpenAlgoOrdersService list open algo orders
type ListOpenAlgoOrdersService struct {
	c      *Client
	symbol *string
}

// Symbol set symbol
func (s *ListOpenAlgoOrdersService) Symbol(symbol string) *ListOpenAlgoOrdersService {
	s.symbol = &symbol
	return s
}

// Do send request
func (s *ListOpenAlgoOrdersService) Do(ctx context.Context, opts ...RequestOption) ([]*AlgoOrder, error) {
	r := &request{
		method:   http.MethodGet,
		endpoint: "/fapi/v1/openAlgoOrders",
		secType:  secTypeSigned,
	}
	if s.symbol != nil {
		r.setParam("symbol", *s.symbol)
	}

	data, _, err := s.c.callAPI(ctx, r, opts...)
	if err != nil {
		return nil, err
	}
	var res []*AlgoOrder
	err = json.Unmarshal(data, &res)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// ListAllAlgoOrdersService list all algo orders
type ListAllAlgoOrdersService struct {
	c         *Client
	symbol    *string
	startTime *int64
	endTime   *int64
	limit     *int
}

// Symbol set symbol
func (s *ListAllAlgoOrdersService) Symbol(symbol string) *ListAllAlgoOrdersService {
	s.symbol = &symbol
	return s
}

// StartTime set startTime
func (s *ListAllAlgoOrdersService) StartTime(startTime int64) *ListAllAlgoOrdersService {
	s.startTime = &startTime
	return s
}

// EndTime set endTime
func (s *ListAllAlgoOrdersService) EndTime(endTime int64) *ListAllAlgoOrdersService {
	s.endTime = &endTime
	return s
}

// Limit set limit
func (s *ListAllAlgoOrdersService) Limit(limit int) *ListAllAlgoOrdersService {
	s.limit = &limit
	return s
}

// Do send request
func (s *ListAllAlgoOrdersService) Do(ctx context.Context, opts ...RequestOption) ([]*AlgoOrder, error) {
	r := &request{
		method:   http.MethodGet,
		endpoint: "/fapi/v1/allAlgoOrders",
		secType:  secTypeSigned,
	}
	if s.symbol != nil {
		r.setParam("symbol", *s.symbol)
	}
	if s.startTime != nil {
		r.setParam("startTime", *s.startTime)
	}
	if s.endTime != nil {
		r.setParam("endTime", *s.endTime)
	}
	if s.limit != nil {
		r.setParam("limit", *s.limit)
	}

	data, _, err := s.c.callAPI(ctx, r, opts...)
	if err != nil {
		return nil, err
	}
	var res []*AlgoOrder
	err = json.Unmarshal(data, &res)
	if err != nil {
		return nil, err
	}
	return res, nil
}
