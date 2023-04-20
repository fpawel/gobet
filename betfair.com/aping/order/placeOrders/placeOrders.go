package placeOrders

import (
	"gobet/betfair.com/aping/appkey"
	"gobet/betfair.com/login"
	"gobet/betfair.com/userSessions"

	"encoding/json"
	"errors"
	"fmt"

	"gobet/betfair.com/aping"
	"gobet/betfair.com/aping/order"
)

// Request заказ ставки
type Request struct {
	User        login.User
	MarketID    string
	SelectionID int
	Side        string
	Price       float64
	Size        float64
}

// placeOrderAPI - сделать ставки
type placeOrderAPI struct {
	MarketID     string                `json:"marketId"`
	Instructions []placeInstructionAPI `json:"instructions"`
}

// placeInstructionAPI - instruction to place request new order
type placeInstructionAPI struct {
	OrderType   string        `json:"orderType"`
	SelectionID int           `json:"selectionId"`
	Side        string        `json:"side"`
	LimitOrder  limitOrderAPI `json:"limitOrder"`
}

// limitOrderAPI - Place request new LIMIT order (simple exchange bet for immediate execution)
type limitOrderAPI struct {
	Size            float64 `json:"size"`
	Price           float64 `json:"price"`
	PersistenceType string  `json:"persistenceType"`
}

type placeOrderReportAPI struct {
	ErrorCode       *string                  `json:"errorCode,omitempty"`
	Status          string                   `json:"status"`
	MarketID        string                   `json:"marketId"`
	PlaceBetReports []PlaceInstructionReport `json:"instructionReports"`
}

type PlaceInstructionReport struct {

	//cause of failure, or null if command succeeds
	ErrorCode *string `json:"errorCode,omitempty"`

	//whether the command succeeded or failed
	Status string `json:"status"`

	//The bet ID of the new bet. May be null on failure.
	BetID *int64 `json:"betId,omitempty"`

	//The date on which the bet was placed
	PlacedDate string `json:"betId,omitempty"`

	//The average price matched
	AveragePriceMatched float32 `json:"averagePriceMatched,omitempty"`

	//The size matched
	SizeMatched float32 `json:"sizeMatched,omitempty"`
}

func (request *Request) GetAPIResponse() (placeBetReports []PlaceInstructionReport, err error) {

	userSessionChanel := make(chan login.Result)
	userSessions.GetUserSession(request.User, userSessionChanel)
	loginResult := <-userSessionChanel
	if loginResult.Error != nil {
		err = fmt.Errorf("login error : %s", loginResult.Error.Error())
		return
	}
	placeOrderRequest := placeOrderAPI{
		MarketID: request.MarketID,
		Instructions: []placeInstructionAPI{
			{
				OrderType:   "LIMIT",
				SelectionID: request.SelectionID,
				Side:        request.Side,
				LimitOrder: limitOrderAPI{
					Size:            request.Size,
					Price:           request.Price,
					PersistenceType: "LAPSE",
				},
			},
		},
	}
	var responseBody []byte
	endpoint := aping.BettingAPI("placeOrders")
	responseBody, err = appkey.GetResponse(loginResult.Token, endpoint, &placeOrderRequest)
	if err != nil {
		err = fmt.Errorf("placeOrders error : %s", err.Error())
		return
	}

	var r placeOrderReportAPI
	err = json.Unmarshal(responseBody, &r)
	if err != nil {
		err = fmt.Errorf("placeOrders unmarshaling response error : %s", err.Error())
		return
	}

	placeBetReports = r.PlaceBetReports

	if r.ErrorCode != nil {
		err = fmt.Errorf("placeOrders error : %s", r.ErrorCode)
		return
	}

	if r.Status != "SUCCESS" {
		err = fmt.Errorf("placeOrders error : status is not SUCCESS : %s", r.Status)
		return
	}

	return
}

func (request *Request) PlaceSingleOrder() (report *order.PlaceOrderReport, err error) {
	var placeBetReports []PlaceInstructionReport
	placeBetReports, err = request.GetAPIResponse()
	if err != nil {
		return
	}

	if len(placeBetReports) == 0 {
		err = errors.New("PlaceSingleOrder : no instruction report")
		return
	}

	p := placeBetReports[0]

	if p.ErrorCode != nil {
		err = fmt.Errorf("PlaceSingleOrder error : %s", p.ErrorCode)
		return
	}

	if p.Status != "SUCCESS" {
		err = fmt.Errorf("PlaceSingleOrder error : status is not SUCCESS : %s", p.Status)
		return
	}

	report = &order.PlaceOrderReport{
		BetID:               *p.BetID,
		AveragePriceMatched: p.AveragePriceMatched,
		SizeMatched:         p.SizeMatched,
	}
	return

}
