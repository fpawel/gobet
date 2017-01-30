package football

import (
	"github.com/user/gobet/betfair.com/aping/client"
	"github.com/user/gobet/betfair.com/aping/client/eventType"
	"log"
)

// Game - данные футбольной игры
type Game struct {
	Live
	EventID  int          `json:"event_id"`
	MarketID int          `json:"market_id"`
	Home     string       `json:"home"`
	Away     string       `json:"away"`
	Event    client.Event `json:"event"`
}

// Odds - котировки основного рынка футбольного матча
type Odds struct {
	Win1  *float64 `json:"win1,omitempty"`
	Win2  *float64 `json:"win2,omitempty"`
	Draw1 *float64 `json:"draw1,omitempty"`
	Draw2 *float64 `json:"draw2,omitempty"`
	Lose1 *float64 `json:"lose1,omitempty"`
	Lose2 *float64 `json:"lose2,omitempty"`
}

// Live - данные футбольной игры, изменяющиеся с течением времени
type Live struct {
	Odds
	Page   int    `json:"page"`
	Order  int    `json:"order"`
	Time   string `json:"time,omitempty"`
	Result string `json:"result,omitempty"`
}

var Events map[int]client.Event

func init() {

	ch := make(chan eventType.ResultGetEvents)
	eventType.GetEvents(1, ch)
	r := <-ch
	if r.Error != nil || r.Result == nil{
		log.Fatalf("error reading footbal events: %v\n", r.Error)
	}

	Events = make(map[int]client.Event)
	for _, event := range *r.Result {
		Events[event.ID] = event
	}
	log.Printf("football: %d events\n", len(Events))

}
