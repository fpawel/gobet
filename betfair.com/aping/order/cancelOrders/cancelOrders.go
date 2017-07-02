package cancelOrders

import (
	"github.com/user/gobet/betfair.com/login"
	"github.com/user/gobet/betfair.com/aping/aping/endpoint"
	"github.com/user/gobet/betfair.com/aping/aping/appkey"
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
	SizeReduction        float64
}

type CancelOrderReport struct{
	SizeCancelled float64
}

func (request *Request) CancelSingleOrder () (report *CancelOrderReport , err error) {
	var instructionReports []CancelInstructionReportAPI
	instructionReports,err = request.GetAPIResponse()
	if err != nil {
		return
	}
	if  len(instructionReports) == 0 {
		err =  errors.New ( "CancelSingleOrder : no instruction report")
		return
	}

	p := instructionReports[0]

	if  p.ErrorCode != nil {
		err = fmt.Errorf( "CancelSingleOrder error : %s", p.ErrorCode)
		return
	}

	if  p.Status != "SUCCESS" {
		err = fmt.Errorf( "CancelSingleOrder error : status is not SUCCESS : %s", p.Status)
		return
	}

	report = &CancelOrderReport{
		SizeCancelled : p.SizeCancelled,
	}
	return
}

func (request *Request) GetAPIResponse () (instructionReports []CancelInstructionReportAPI, err error) {

	userSessionChanel := make(chan login.Result)
	userSessions.GetUserSession(request.User, userSessionChanel)
	loginResult := <-userSessionChanel
	if loginResult.Error != nil {
		err = fmt.Errorf( "cancelOrders : login error : %s", loginResult.Error.Error())
		return
	}
	placeOrderRequest := cancelOrderAPI{
		MarketID: request.MarketID,
		Instructions: []CancelInstructionAPI{
			{
				BetId:         request.BetID,
				SizeReduction: request.SizeReduction,
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

	if  r.Status != "SUCCESS" {
		err = fmt.Errorf( "cancelOrders error : status is not SUCCESS : %s", r.Status)
		return
	}

	return
}

type CancelInstructionAPI struct {
	BetId  int64 `json:"betId"`
	// If supplied then this is a partial cancel.  Should be set to 'null' if no size reduction is required
	SizeReduction float64 `json:"sizeReduction"`
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

	SizeCancelled float64 `json:"sizeCancelled"`

	CancelledDate string `json:"cancelledDate"`
}