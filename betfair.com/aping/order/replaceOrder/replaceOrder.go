package replaceOrder

import (
	"github.com/user/gobet/betfair.com/aping/order/cancelOrders"
	"github.com/user/gobet/betfair.com/aping/order/placeOrders"
	"github.com/user/gobet/betfair.com/login"
	"github.com/user/gobet/betfair.com/userSessions"
	"fmt"
	"github.com/user/gobet/betfair.com/aping"
	"github.com/user/gobet/betfair.com/aping/appkey"
	"encoding/json"
	"errors"
	"github.com/user/gobet/betfair.com/aping/order"
)

type Request struct {
	BetID int64
	NewPrice float64 `json:"newPrice"`
	User login.User
	MarketID string
}

func (request *Request) ReplaceSingleOrder() (report *order.PlaceOrderReport, err error) {

	var instructionReports []ReplaceInstructionReport
	instructionReports,err = request.GetAPIResponse()

	p := instructionReports[0].PlaceInstructionReport

	if  p.ErrorCode != nil {
		err = fmt.Errorf( "ReplaceSingleOrder error : %s", p.ErrorCode)
		return
	}

	if  p.Status != "SUCCESS" {
		err = fmt.Errorf( "ReplaceSingleOrder error : status is not SUCCESS : %s", p.Status)
		return
	}

	if  p.BetID == nil {
		err = errors.New( "ReplaceSingleOrder error : no Bet ID")
		return
	}

	report = &order.PlaceOrderReport{
		BetID : *p.BetID,
		AveragePriceMatched : p.AveragePriceMatched,
		SizeMatched : p.SizeMatched,
	}

	return
}

func (request *Request) GetAPIResponse()(instructionReports []ReplaceInstructionReport, err error){
	userSessionChanel := make(chan login.Result)
	userSessions.GetUserSession(request.User, userSessionChanel)
	loginResult := <-userSessionChanel
	if loginResult.Error != nil {
		err = fmt.Errorf( "replaceOrder : login error : %s", loginResult.Error.Error())
		return
	}

	apiRequest := ReplaceRequest{
		MarketID : request.MarketID,
		Instructions : []ReplaceInstruction{
			ReplaceInstruction{
				BetID : request.BetID,
				NewPrice: request.NewPrice,
			},
		},
	}

	var responseBody []byte
	endpoint := aping.BettingAPI("replaceOrders")

	responseBody, err = appkey.GetResponse(loginResult.Token, endpoint, &apiRequest)
	if err != nil {
		err = fmt.Errorf( "replaceOrder error : %s", err.Error())
		return
	}

	var r ReplaceExecutionReport
	err = json.Unmarshal(responseBody, &r)
	if err != nil {
		err = fmt.Errorf( "replaceOrder unmarshaling response error : %s", err.Error())
		return
	}

	if  r.ErrorCode != nil {
		err = fmt.Errorf( "replaceOrder error : %s", r.ErrorCode)
		return
	}

	if  r.Status != "SUCCESS" {
		err = fmt.Errorf( "replaceOrder error : status : %s", r.Status)
		return
	}

	if  len(r.InstructionReports) == 0 {
		err =  errors.New ( "replaceOrder : no instruction report")
		return
	}

	instructionReports = r.InstructionReports
	return
}



type ReplaceInstruction struct {
	// Unique identifier for the bet
	BetID int64 `json:"betId"`

	// The price to replace the bet at
	NewPrice float64 `json:"newPrice"`
}

type ReplaceRequest struct {
	MarketID     string               `json:"marketId"`
	Instructions []ReplaceInstruction `json:"instructions"`
}

type ReplaceInstructionReport struct {

	// whether the command succeeded or failed
	Status string `json:"status"`

	//cause of failure, or null if command succeeds
	ErrorCode *string `json:"errorCode,omitempty"`

	// Cancelation report for the original order
	CancelInstructionReport cancelOrders.CancelInstructionAPI `json:"cancelInstructionReport"`

	//Placement report for the new order
	PlaceInstructionReport placeOrders.PlaceInstructionReport `json:"placeInstructionReport"`
}

type ReplaceExecutionReport struct {
	Status             string                     `json:"status"`
	ErrorCode          *string                     `json:"errorCode,omitempty"`
	MarketID           string                     `json:"marketId"`
	InstructionReports []ReplaceInstructionReport `json:"instructionReports"`
}


