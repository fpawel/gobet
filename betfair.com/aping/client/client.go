package client

type EventType struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	MarketCount int    `json:"market_count"`
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

	Venue string `json:"Venue"`

	// The scheduled start date and time of the event.
	// This is Europe/London (GMT) by default
	OpenDate string `json:"open_date"`

	// Count of markets associated with this event
	MarketCount int `json:"marketCount,omitempty"`
}

type MarketBase struct {

	//  The name of the market
	Name string `json:"marketName"`

	// The total amount of money matched on the market
	TotalMatched float64 `json:"totalMatched,omitempty"`

	// The runners (selections) contained in the market
	Runners []struct {
		// The unique id for the selection
		ID int `json:"selectionId"`
		// The name of the runner
		Name string `json:"runnerName"`
	} `json:"runners,omitempty"`
	// The competition the market is contained within. Usually only applies to Football competitions
	Competition string `json:"competition,omitempty"`
}

type Market struct {
	MarketBase

	// The unique identifier for the market. MarketId's are prefixed with '1.' or '2.' 1. = UK Exchange 2. = AUS Exchange."
	ID int `json:"marketId"`
}
