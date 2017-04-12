package cancelOrders

import (
	"github.com/user/gobet/betfair.com/login"
	"github.com/user/gobet/betfair.com/aping/client/endpoint"
	"github.com/user/gobet/betfair.com/aping/client/appkey"
	"fmt"
	"encoding/json"
	"errors"
	"github.com/user/gobet/betfair.com/userSessions"
)

// Request заказ ставки
type Request struct {
	User        login.User
	MarketID    string
	BetID int64
	SizeReduction        float32
}

func GetAPIResponse(a *Request) (instructionReports []CancelInstructionReportAPI, err error) {

	userSessionChanel := make(chan login.Result)
	userSessions.GetUserSession(a.User, userSessionChanel)
	loginResult := <-userSessionChanel
	if loginResult.Error != nil {
		err = fmt.Errorf( "cancelOrders : login error : %s", loginResult.Error.Error())
		return
	}
	placeOrderRequest := cancelOrderAPI{
		MarketID: a.MarketID,
		Instructions: []CancelInstructionAPI{
			{
				BetId:a.BetID,
				SizeReduction: a.SizeReduction,
			},
		},
	}
	var responseBody []byte
	endpoint := endpoint.BettingAPI("cancelOrders")
	responseBody, err = appkey.GetResponse(loginResult.Token, endpoint, &placeOrderRequest)
	if err != nil {
		err = fmt.Errorf( "cancelOrders error : %s", err.Error())
		return
	}

	var r cancelOrderReportAPI
	err = json.Unmarshal(responseBody, &r)
	if err != nil {
		err = fmt.Errorf( "cancelOrders unmarshaling response error : %s", err.Error())
		return
	}

	instructionReports = r.InstructionReports

	if  r.ErrorCode != nil {
		err = fmt.Errorf( "cancelOrders error : %s", r.ErrorCode)
		return
	}

	if  len(r.InstructionReports) == 0 {
		err =  errors.New ( "cancelOrders : no instruction report")
		return
	}
	return
}

type CancelInstructionAPI struct {
	BetId  int64 `json:"betId"`
	// If supplied then this is a partial cancel.  Should be set to 'null' if no size reduction is required
	SizeReduction float32 `json:"sizeReduction"`
}

type cancelOrderAPI struct{
	MarketID string `json:"marketId"`
	Instructions [] CancelInstructionAPI  `json:"instructions"`
}

type cancelOrderReportAPI struct {
	// whether the command succeeded or failed
	Status string `json:"status"`

	//cause of failure, or null if command succeeds
	ErrorCode *string `json:"errorCode,omitempty"`

	MarketId string `json:"marketId"`

	InstructionReports []CancelInstructionReportAPI `json:"instructionReports"`
}

type CancelInstructionReportAPI struct{
	// whether the command succeeded or failed
	Status string `json:"status"`

	// cause of failure, or null if command succeeds
	ErrorCode *string `json:"errorCode,omitempty"`

	// The instruction that was requested
	Instruction  CancelInstructionAPI `json:"instruction"`

	SizeCancelled float32 `json:"sizeCancelled"`

	CancelledDate string `json:"cancelledDate"`
}