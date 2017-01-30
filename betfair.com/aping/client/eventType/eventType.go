package eventType

import (
	"encoding/json"
	"fmt"
	"github.com/user/gobet/betfair.com/aping/client"
	"github.com/user/gobet/betfair.com/aping/client/appkey"
	"github.com/user/gobet/betfair.com/aping/client/endpoint"
	"log"
	"strconv"
	"sync"
)

type Events []client.Event

var muAwaiters sync.RWMutex
var awaiters map[int][]chan<- ResultGetEvents

type ResultGetEvents struct {
	Result *Events `json:"result,omitempty"`
	Error  error  `json:"error,omitempty"`
}

func GetEvents(eventTypeID int, ch chan<- ResultGetEvents) {

	muAwaiters.RLock()
	if xs, yes := awaiters[eventTypeID]; yes {
		muAwaiters.RUnlock()
		muAwaiters.Lock()
		awaiters[eventTypeID] = append(xs, ch)
		muAwaiters.Unlock()
		return
	}
	muAwaiters.RUnlock()

	muAwaiters.Lock()
	awaiters[eventTypeID] = []chan<- ResultGetEvents{ch}
	muAwaiters.Unlock()

	go func() {
		events, err := readEvents(eventTypeID)
		result := ResultGetEvents{&events, err}

		muAwaiters.RLock()
		thisAwaiters, ok := awaiters[eventTypeID]
		if !ok || len(thisAwaiters) == 0 {
			log.Fatalln("awaiters list is empty")
		}
		muAwaiters.RUnlock()

		for _, ch := range thisAwaiters {
			ch <- result
		}

		muAwaiters.Lock()
		delete(awaiters, eventTypeID)
		muAwaiters.Unlock()

	}()

}

func readEvents(eventTypeID int) (events Events, err error) {
	var reqParams struct {
		Locale string `json:"locale"`
		Filter struct {
			EventTypeIDs []int `json:"eventTypeIds"`
		} `json:"filter"`
	}
	reqParams.Locale = "ru"
	reqParams.Filter.EventTypeIDs = []int{eventTypeID}
	ep := endpoint.BettingAPI("listEvents")
	responseBody, err := appkey.GetResponse(ep, &reqParams)
	if err != nil {
		return
	}
	var xs struct {
		R []struct {
			MarketCount int `json:"marketCount"`
			X           struct {
				ID          string `json:"id"`
				Name        string `json:"name"`
				CountryCode string `json:"countryCode,omitempty"`
				Timezone    string `json:"timezone,omitempty"`
				Venue       string `json:"venue,omitempty"`
				OpenDate    string `json:"openDate,omitempty"`
			} `json:"event"`
		} `json:"result"`
	}
	err = json.Unmarshal(responseBody, &xs)
	if err != nil {
		return
	}
	for _, x := range xs.R {
		var y client.Event
		y.ID, err = strconv.Atoi(x.X.ID)
		if err != nil {
			err = fmt.Errorf("wrong event id %v: %v", x, err)
			return
		}
		y.Name = x.X.Name
		y.CountryCode = x.X.CountryCode
		y.Timezone = x.X.Timezone
		y.Venue = x.X.Venue
		y.OpenDate = x.X.OpenDate
		y.MarketCount = x.MarketCount
		events = append(events, y)
	}
	return
}

func init() {
	awaiters = make(map[int][]chan<- ResultGetEvents)
}
