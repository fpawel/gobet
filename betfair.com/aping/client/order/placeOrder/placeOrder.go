package placeOrder

import (
	"github.com/user/gobet/betfair.com/aping/client/appkey"
	"github.com/user/gobet/betfair.com/aping/client/endpoint"
	"github.com/user/gobet/betfair.com/login"
	"github.com/user/gobet/betfair.com/userSessions"

	"encoding/json"
	"fmt"
	"errors"
)

// Request заказ ставки
type Request struct {
	User        login.User
	MarketID    string
	SelectionID int
	Side        string
	Price       float32
	Size        float32
}

// placeOrderAPI - сделать ставки
type placeOrderAPI struct {
	MarketID     string                `json:"marketId"`
	Instructions []placeInstructionAPI `json:"instructions"`
}

// placeInstructionAPI - instruction to place a new order
type placeInstructionAPI struct {
	OrderType   string        `json:"orderType"`
	SelectionID int           `json:"selectionId"`
	Side        string        `json:"side"`
	LimitOrder  limitOrderAPI `json:"limitOrder"`
}

// limitOrderAPI - Place a new LIMIT order (simple exchange bet for immediate execution)
type limitOrderAPI struct {
	Size            float32 `json:"size"`
	Price           float32 `json:"price"`
	PersistenceType string  `json:"persistenceType"`
}

type placeOrderReportAPI struct {
	ErrorCode       *string                      `json:"errorCode,omitempty"`
	Status          string                      `json:"status"`
	MarketID        string                      `json:"marketId"`
	PlaceBetReports []PlaceBetReportAPI `json:"instructionReports"`
}

type PlaceBetReportAPI struct {

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


func GetAPIResponse(a *Request) (placeBetReports []PlaceBetReportAPI, err error) {

	userSessionChanel := make(chan login.Result)
	userSessions.GetUserSession(a.User, userSessionChanel)
	loginResult := <-userSessionChanel
	if loginResult.Error {
		err = fmt.Errorf( "login error : %s", loginResult.Error.Error())
		return
	}
	placeOrderRequest := placeOrderAPI{
		MarketID: a.MarketID,
		Instructions: []placeInstructionAPI{
			{
				OrderType:   "LIMIT",
				SelectionID: a.SelectionID,
				Side:        a.Side,
				LimitOrder: limitOrderAPI{
					Size:            a.Size,
					Price:           a.Price,
					PersistenceType: "LAPSE",
				},
			},
		},
	}
	var responseBody []byte
	endpoint := endpoint.BettingAPI("placeOrders")
	responseBody, err = appkey.GetResponse(loginResult.Token, endpoint, &placeOrderRequest)
	if err != nil {
		err = fmt.Errorf( "placeOrders error : %s", err.Error())
		return
	}

	var r placeOrderReportAPI
	err = json.Unmarshal(responseBody, &r)
	if err != nil {
		err = fmt.Errorf( "placeOrders unmarshaling response error : %s", err.Error())
		return
	}

	placeBetReports = r.PlaceBetReports

	if  r.ErrorCode != nil {
		err = fmt.Errorf( "placeOrders error : %s", r.ErrorCode)
		return
	}

	if  len(r.PlaceBetReports) == 0 {
		err =  errors.New ( "placeOrders : no instruction report")
		return
	}
	return
}

