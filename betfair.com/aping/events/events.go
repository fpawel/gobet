package events

import (
	"encoding/json"
	"fmt"
	"github.com/user/gobet/betfair.com/aping"
	"github.com/user/gobet/betfair.com/aping/appkey"
	"log"
	"strconv"
	"sync"

	"time"
)

type Events []aping.Event

var muAwaiters sync.RWMutex
var awaiters map[int][]chan<- Result = make(map[int][]chan<- Result)

type listEventsTime struct {
	listEvents Events
	time       time.Time
}

var muCache sync.RWMutex
var cache map[int]listEventsTime = make(map[int]listEventsTime)

type Result struct {
	Events Events `json:"result,omitempty"`
	Error  error  `json:"error,omitempty"`
}

// получить список событий из cache
func getEventsFromCache(eventTypeID int) (events Events) {
	muCache.RLock()
	defer muCache.RUnlock()
	if x, f := cache[eventTypeID]; f && time.Since(x.time) < 10*time.Minute {
		events = append(x.listEvents[:])
	}
	return
}

func doReadEvents(eventTypeID int) {
	result := Result{}
	result.Events, result.Error = getResponse(eventTypeID)
	log.Printf("read events %d: %v\n", eventTypeID, result.Error)

	if result.Error == nil {
		x := listEventsTime{}
		x.time = time.Now()
		x.listEvents = append(result.Events)

		muCache.Lock()
		cache[eventTypeID] = x
		muCache.Unlock()
	}

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

}

func ClearCache(eventTypeID int) {
	muCache.Lock()
	defer muCache.Unlock()
	delete(cache, eventTypeID)
}

func Get(eventTypeID int, ch chan<- Result) {

	// если удалось получить список событий из cache, записать список событий в канал
	if inCache := getEventsFromCache(eventTypeID); inCache != nil {
		go func() {
			ch <- Result{inCache, nil}
		}()
		return
	}

	// если список ожидающих каналов awaiters[eventTypeID] не пуст,
	muAwaiters.RLock()
	if xs, yes := awaiters[eventTypeID]; yes {
		// в настоящий момент другой поток получает список событий в doReadEvents(eventTypeID).
		muAwaiters.RUnlock()
		muAwaiters.Lock()
		// добавить ещё один канал ch к списку awaiters[eventTypeID] и вернуть управление
		awaiters[eventTypeID] = append(xs, ch)
		muAwaiters.Unlock()
		return
	}
	muAwaiters.RUnlock()

	// внечсти канал ch в список awaiters[eventTypeID]
	muAwaiters.Lock()
	awaiters[eventTypeID] = []chan<- Result{ch}
	muAwaiters.Unlock()

	go doReadEvents(eventTypeID)

	return
}

func getResponse(eventTypeID int) (events Events, err error) {

	var reqParams struct {
		Locale string `json:"locale"`
		Filter struct {
			EventTypeIDs []int `json:"eventTypeIds"`
		} `json:"filter"`
	}
	reqParams.Locale = "ru"
	reqParams.Filter.EventTypeIDs = []int{eventTypeID}
	ep := aping.BettingAPI("listEvents")
	responseBody, err := appkey.GetResponseWithAdminLogin(ep, &reqParams)
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
		var y aping.Event
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
	log.Printf("readed events %d", eventTypeID)




	return
}
