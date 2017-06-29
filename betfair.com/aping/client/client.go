package client

import (
	"time"
)

type EventType struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	MarketCount int    `json:"market_count,omitempty"`
}

type Event struct {

	// The unique id for the event
	ID int `json:"id"`

	// The name of the event
	Name string `json:"name"`

	// The ISO-2 code for the event.
	// A list of ISO-2 codes is available via
	// http://en.wikipedia.org/wiki/ISO_3166-1_alpha-2
	CountryCode string `json:"country_code"`

	// This is timezone in which the event is taking place./
	Timezone string `json:"time_zone"`

	Venue string `json:"venue"`

	// The scheduled start date and time of the event.
	// This is Europe/London (GMT) by default
	OpenDate string `json:"open_date"`

	// Count of markets associated with this event
	MarketCount int `json:"market_count,omitempty"`

	EventType *EventType `json:"event_type,omitempty"`

	Markets []Market `json:"markets,omitempty"`
}

type Market struct {

	//The unique identifier for the market. MarketId's are prefixed with '1.' or '2.' 1. = UK Exchange 2. = AUS Exchange.
	ID string `json:"marketId"`

	//  The name of the market
	Name string `json:"marketName,omitempty"`

	//True if the data returned by listMarketBook will be delayed.
	// The data may be delayed because you are not logged in with a funded account or you are using an Application Key
	// that does not allow up to date data.
	IsMarketDataDelayed bool `json:"isMarketDataDelayed,omitempty"`

	//The status of the market, for example ACTIVE, SUSPENDED, SETTLED, etc.
	Status string `json:"status,omitempty"`

	// The number of seconds an order is held until it is submitted into the market.
	// Orders are usually delayed when the market is in-play
	BetDelay int `json:"betDelay,omitempty"`

	// True if the market starting price has been reconciled
	BspReconciled bool `json:"bspReconciled,omitempty"`

	//If false, runners may be added to the market
	Complete *bool `json:"complete,omitempty"`

	// True if the market is currently in play
	Inplay bool `json:"inplay,omitempty"`

	//The number of selections that could be settled as winners
	NumberOfWinners int `json:"numberOfWinners,omitempty"`

	//The number of runners in the market
	NumberOfRunners int `json:"numberOfRunners,omitempty"`

	//The number of runners that are currently active. An active runner is a selection available for betting
	NumberOfActiveRunners int `json:"numberOfActiveRunners,omitempty"`

	//The most recent time an order was executed
	LastMatchTime string `json:"lastMatchTime,omitempty"`

	//The total amount matched
	TotalMatched float64 `json:"totalMatched,omitempty"`

	//The total amount of orders that remain unmatched
	TotalAvailable float64 `json:"totalAvailable,omitempty"`

	//True if cross matching is enabled for this market.
	CrossMatching bool `json:"crossMatching,omitempty"`

	//True if runners in the market can be voided
	RunnersVoidable bool `json:"runnersVoidable,omitempty"`

	//The version of the market. The version increments whenever the market status changes,
	// for example, turning in-play, or suspended when a goal score.
	Version int `json:"version,omitempty"`

	//Information about the runners (selections) in the market.
	Runners []Runner `json:"runners,omitempty"`
}

///The dynamic data about runners in a market
type Runner struct {

	// The unique id of the runner (selection)
	ID int `json:"selectionId"`

	// The name of the runner
	Name string `json:"runnerName,omitempty"`

	// The status of the selection (i.e., ACTIVE, REMOVED, WINNER, LOSER, HIDDEN)
	// Runner status information is available for 90 days following market settlement.
	Status string `json:"status,omitempty"`

	// The adjustment factor applied if the selection is removed
	AdjustmentFactor float64 `json:"adjustmentFactor,omitempty"`

	// The price of the most recent bet matched on this selection
	LastPriceTraded float64 `json:"lastPriceTraded,omitempty"`

	// The total amount matched on this runner
	TotalMatched float64 `json:"totalMatched,omitempty"`

	// If date and time the runner was removed
	RemovalDate string `json:"removalDate,omitempty"`

	// The Exchange prices available for this runner
	ExchangePrices *ExchangePrices `json:"ex,omitempty"`

	//The sort priority of this runner
	SortPriority int `json:"sortPriority,omitempty"`

	// The handicap.  Enter the specific handicap value (returned by RUNNER in listMaketBook)
	// if the market is an Asian handicap market.
	// Handicap float64 `json:"handicap,omitempty"`

	// The BSP related prices for this runner
	//sp : StartingPrices option

	// List of orders in the market
	//orders : Order list
	// List of matches (i.e, orders that have been fully or partially executed)
	//matches : Match list
}

type ExchangePrices struct {
	AvailableToBack []Odd `json:"availableToBack"`
	AvailableToLay  []Odd `json:"availableToLay"`
	TradedVolume    []Odd `json:"tradedVolume"`
}

type Odd struct {
	Price float64 `json:"price"`
	Size  float64 `json:"size"`
}

func (x *Runner) GetOdd(side string, index int) *Odd {
	if x.ExchangePrices == nil {
		return nil
	}
	xs := x.ExchangePrices.AvailableToBack
	if side == "LAY" {
		xs = x.ExchangePrices.AvailableToLay
	}
	if index >= len(xs) {
		return nil
	}
	return &xs[index]
}

func (market *Market) Copy(copyMarket *Market) {
	*copyMarket = *market
	copyMarket.Runners = make([]Runner, len(market.Runners))
	if copy(copyMarket.Runners, market.Runners) != len(market.Runners) {
		panic("wtf?")
	}
}

func (x *Event) Copy(theCopy *Event) {
	*theCopy = *x
	theCopy.Markets = make([]Market, len(x.Markets))
	for n := range x.Markets {
		x.Markets[n].Copy(&theCopy.Markets[n])
	}
}

func NewEvent(id int, name string) Event {
	return Event{
		ID:          id,
		Name:        name,
		CountryCode: "",
		OpenDate:    time.Now().String(),
		Timezone:    "",
		Venue:       "",
	}
}
