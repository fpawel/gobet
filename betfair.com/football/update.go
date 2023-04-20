package football

import (
	"encoding/json"
	"hash/fnv"
	"log"

	"gobet/betfair.com/aping"
	"strconv"
)

// MatchChanges содержит изменения в данных футбольной игры на стороне сервера,
// расчитанных по данным футбольной игры на стороне клиента
type MatchChanges struct {
	EventID int     `json:"event_id"`
	Page    *int    `json:"page,omitempty"`
	Order   *int    `json:"order,omitempty"`
	Time    *string `json:"time,omitempty"`
	Result  *string `json:"result,omitempty"`

	Win1  **float64 `json:"win1,omitempty"`
	Win2  **float64 `json:"win2,omitempty"`
	Draw1 **float64 `json:"draw1,omitempty"`
	Draw2 **float64 `json:"draw2,omitempty"`
	Lose1 **float64 `json:"lose1,omitempty"`
	Lose2 **float64 `json:"lose2,omitempty"`
}

// MatchesListChanges содержит данные об изменения в списке игр на стороне сервера,
// расчитанных по списку игр на стороне клиента
type MatchesListChanges struct {
	Inplay  []Match        `json:"inplay,omitempty"`
	Outplay []int          `json:"outplay,omitempty"`
	Changes []MatchChanges `json:"game_changes,omitempty"`
	Events  []aping.Event  `json:"events,omitempty"`
}

func (x *MatchesListChanges) isEmpty() bool {
	return len(x.Inplay) == 0 && len(x.Outplay) == 0 && len(x.Changes) == 0
}

func setp1(x *float64, y *float64, z ***float64) {
	eq := x == nil && y == nil || x != nil && y != nil && *x == *y
	if !eq {
		*z = &y
	}
}

func (x *MatchesListChanges) GetHashCode() string {
	fnv32a := fnv.New64a()
	bytes, err := json.Marshal(x)
	if err != nil {
		log.Fatal("json.Marshal MatchesListChanges")
	}
	fnv32a.Write(bytes)

	return strconv.FormatUint(fnv32a.Sum64(), 16)
}

func (x *MatchesListChanges) SetInplayEvents(events []aping.Event) {
	inplays := make(map[int]struct{})
	for _, y := range x.Inplay {
		inplays[y.EventID] = struct{}{}
	}
	for _, y := range events {
		if _, ok := inplays[y.ID]; ok {
			x.Events = append(x.Events, y)
		}
	}
}

func NewMatchesListChanges(prev []Match, next []Match) *MatchesListChanges {
	x := MatchesListChanges{}
	mprev := make(map[int]Match)
	mnext := make(map[int]Match)

	for _, game := range prev {
		mprev[game.EventID] = game
	}

	for _, game := range next {
		mnext[game.EventID] = game

		if _, ok := mprev[game.EventID]; !ok {
			x.Inplay = append(x.Inplay, game)
		}
	}

	for eventid, gamePrev := range mprev {
		gameNext, ok := mnext[eventid]
		if !ok {
			x.Outplay = append(x.Outplay, eventid)
		} else {

			if gamePrev.Live != gameNext.Live {
				changes := difference(&gamePrev.Live, &gameNext.Live)
				if changes != nil {
					changes.EventID = eventid
					x.Changes = append(x.Changes, *changes)
				}
			}
		}
	}

	if x.isEmpty() {
		return nil
	}

	return &x
}

func difference(x *Live, y *Live) *MatchChanges {

	r := MatchChanges{}
	if x.Time != y.Time {
		r.Time = &y.Time
	}
	if x.Result != y.Result {
		r.Result = &y.Result
	}
	if x.Order != y.Order {
		r.Order = &y.Order
	}
	if x.Page != y.Page {
		r.Page = &y.Page
	}
	setp1(x.Win1, y.Win1, &r.Win1)
	setp1(x.Win2, y.Win2, &r.Win2)
	setp1(x.Draw1, y.Draw1, &r.Draw1)
	setp1(x.Draw2, y.Draw2, &r.Draw2)
	setp1(x.Lose1, y.Lose1, &r.Lose1)
	setp1(x.Lose2, y.Lose2, &r.Lose2)

	if r.Time != nil || r.Result != nil ||
		r.Order != nil || r.Page != nil ||
		r.Win1 != nil || r.Win2 != nil ||
		r.Draw1 != nil || r.Draw2 != nil ||
		r.Lose1 != nil || r.Lose2 != nil {
		return &r
	}
	return nil
}
