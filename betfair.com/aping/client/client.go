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
