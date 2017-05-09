package replaceOrder

import (
	"github.com/user/gobet/betfair.com/aping/client/order/cancelOrders"
	"github.com/user/gobet/betfair.com/aping/client/order/placeOrders"
)

type ReplaceInstruction struct {
	// Unique identifier for the bet
	BetId int64 `json:"betId"`

	// The price to replace the bet at
	NewPrice float32 `json:"newPrice"`
}

type ReplaceRequest struct {
	MarketID     string               `json:"marketId"`
	instructions []ReplaceInstruction `json:"instructions"`
}

type ReplaceInstructionReport struct {

	// whether the command succeeded or failed
	Status string `json:"status"`

	//cause of failure, or null if command succeeds
	ErrorCode string `json:"errorCode"`

	// Cancelation report for the original order
	CancelInstructionReport cancelOrders.CancelInstructionAPI `json:"cancelInstructionReport"`

	//Placement report for the new order
	PlaceInstructionReport placeOrders.PlaceInstructionReport `json:"placeInstructionReport"`
}

type ReplaceExecutionReport struct {
	// Echo of the customerRef if passed
	CustomerRef string `json:"placeInstructionReport,ommitempty"`
	Status      string `json:"status"`
	ErrorCode   string `json:"errorCode"`

	MarketID string `json:"marketId"`

	InstructionReports []ReplaceInstructionReport `json:"instructionReports"`
}
