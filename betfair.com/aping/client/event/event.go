package event

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/user/gobet/betfair.com/aping/client"
	"github.com/user/gobet/betfair.com/aping/client/appkey"
	"github.com/user/gobet/betfair.com/aping/client/endpoint"
)

func Get(eventID int, ch chan<- Result) {
	readerInstance.get(eventID, ch)
}

var readerInstance = newReader()

type reader struct {
	muAwaiters *sync.RWMutex
	awaiters   map[int][]chan<- Result
	muCache    *sync.RWMutex
	cache      map[int]cacheItemType
}

func newReader() (x *reader) {
	x = new(reader)
	x.muAwaiters = new(sync.RWMutex)
	x.awaiters = make(map[int][]chan<- Result)
	x.cache = make(map[int]cacheItemType)
	x.muCache = new(sync.RWMutex)
	return
}

type cacheItemType struct {
	event *client.Event
	time  time.Time
}

type Result struct {
	Event *client.Event
	Error error
}

// получить список рынков из cache
func (reader *reader) getFromCache(eventID int) (result *client.Event) {
	reader.muCache.RLock()
	defer reader.muCache.RUnlock()
	if incache, f := reader.cache[eventID]; f && time.Since(incache.time) < time.Minute {
		result = &client.Event{}
		incache.event.Copy(result)
	}
	return
}

func (reader *reader) performReadEvent(eventID int) {

	event, err := readEvent(eventID)
	log.Printf("read event %d: %v\n", eventID, err)

	if err == nil {
		x := cacheItemType{event, time.Now()}
		reader.muCache.Lock()
		reader.cache[eventID] = x
		reader.muCache.Unlock()
	}

	reader.muAwaiters.Lock()
	thisAwaiters, ok := reader.awaiters[eventID]
	delete(reader.awaiters, eventID)
	reader.muAwaiters.Unlock()

	if !ok || len(thisAwaiters) == 0 {
		log.Fatalln("awaiters list is empty")
	}

	for _, ch := range thisAwaiters {
		ch <- Result{event, err}
	}
}

func (reader *reader) clear(eventID int) {
	reader.muCache.Lock()
	defer reader.muCache.Unlock()
	delete(reader.cache, eventID)
}

func (reader *reader) get(eventID int, ch chan<- Result) {

	// если удалось получить список рынков из cache, записать список рынков в канал
	if cachedContent := reader.getFromCache(eventID); cachedContent != nil {
		go func() {
			ch <- Result{cachedContent, nil}
		}()
		return
	}

	// если список ожидающих каналов awaiters[eventID] не пуст,
	reader.muAwaiters.RLock()
	if xs, yes := reader.awaiters[eventID]; yes {
		// в настоящий момент другой поток получает список событий в doReadEvents(eventTypeID).
		reader.muAwaiters.RUnlock()
		reader.muAwaiters.Lock()
		// добавить ещё один канал ch к списку awaiters[eventID] и вернуть управление
		reader.awaiters[eventID] = append(xs, ch)
		reader.muAwaiters.Unlock()
		return
	}
	reader.muAwaiters.RUnlock()

	// внечсти канал ch в список awaiters[eventID]
	reader.muAwaiters.Lock()
	reader.awaiters[eventID] = []chan<- Result{ch}
	reader.muAwaiters.Unlock()

	go reader.performReadEvent(eventID)

	return
}

/*
func extractMarketID(s string) (id int, err error) {
	xs := strings.Split(s, "1.")
	if len(xs) == 2 {
		id, err = strconv.Atoi(xs[1])
	} else {
		err = fmt.Errorf("%v is not valid market id", s)
	}
	return

}
*/

func readMarkets(eventID int) (markets []client.Market, err error) {
	var reqParams struct {
		Locale           string   `json:"locale"`
		MarketProjection []string `json:"marketProjection"`
		Filter           struct {
			EventIDs []int `json:"eventIds"`
		} `json:"filter"`
		MaxResults int `json:"maxResults"`
	}

	reqParams.Locale = "ru"
	reqParams.Filter.EventIDs = []int{eventID}
	reqParams.MarketProjection = []string{"RUNNER_DESCRIPTION"}
	reqParams.MaxResults = 1000

	ep := endpoint.BettingAPI("listMarketCatalogue")

	responseBody, err := appkey.GetResponse(ep, &reqParams)
	if err != nil {
		return
	}
	var responseData struct {
		Result [] client.Market `json:"result"`
	}
	err = json.Unmarshal(responseBody, &responseData)
	if err != nil {
		err = fmt.Errorf("%q, %q", err, string(responseBody))
		return
	}

	for _, market := range responseData.Result {
		markets = append(markets, market)
	}
	return
}

func readMarketEvent(event *client.Event) error {
	var reqParams struct {
		Locale           string   `json:"locale"`
		MarketProjection []string `json:"marketProjection"`
		Filter           struct {
			EventIDs []int `json:"eventIds"`
		} `json:"filter"`
		MaxResults int `json:"maxResults"`
	}

	reqParams.Locale = "ru"
	reqParams.Filter.EventIDs = []int{event.ID}
	//reqParams.Filter.MarketIDs = []int{marketID}
	reqParams.MarketProjection = []string{"EVENT", "EVENT_TYPE"}
	reqParams.MaxResults = 1

	ep := endpoint.BettingAPI("listMarketCatalogue")

	responseBody, err := appkey.GetResponse(ep, &reqParams)
	if err != nil {
		return err
	}
	var xs struct {
		R []struct {
			EventType struct {
				ID          string `json:"id"`
				Name        string `json:"name"`
			} `json:"eventType,omitempty"`
			Event struct {
				ID          string `json:"id"`
				Name        string `json:"name"`
				CountryCode string `json:"countryCode"`
				Timezone    string `json:"timezone"`
				Venue       string `json:"venue"`
				OpenDate    string `json:"opendate"`
			} `json:"event,omitempty"`
		} `json:"result"`
	}
	err = json.Unmarshal(responseBody, &xs)
	if err != nil {
		return fmt.Errorf("%q, %q", err, string(responseBody))
	}

	if len(xs.R) != 1 {
		return fmt.Errorf("unexpected markets count %d, expected 1, %s", len(xs.R), string(responseBody))
	}
	x := xs.R[0]
	var eventTypeID int
	eventTypeID, err = strconv.Atoi(x.EventType.ID)
	if err != nil {
		return fmt.Errorf("cant convert `eventTypeID`=%v to int: %v, %v", x.EventType.ID, err, string(responseBody))
	}

	event.EventType = &client.EventType{
		ID:          eventTypeID,
		Name:        strings.Trim(x.EventType.Name, " ") }

	e := x.Event

	event.Name = strings.Trim(e.Name, " ")
	event.CountryCode = e.CountryCode
	event.Timezone = e.Timezone
	event.Venue = e.Venue
	event.OpenDate = e.OpenDate

	return nil
}

func readEvent(eventID int) (event *client.Event, err error) {
	var markets []client.Market

	markets, err = readMarkets(eventID)
	if err != nil {
		return
	}
	if len(markets) == 0 {
		err = errors.New("no markets")
		return
	}

	event = &client.Event{ID: eventID}
	err = readMarketEvent(event)
	if err != nil {
		event = nil
		err = fmt.Errorf("%s, %d markets readed", err.Error(), len(markets))
		return
	}
	event.Markets = markets
	return
}
