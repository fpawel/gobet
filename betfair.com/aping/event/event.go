package event

import (
	"encoding/json"

	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/user/gobet/betfair.com/aping"
	"github.com/user/gobet/betfair.com/aping/appkey"
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
	event *aping.Event
	time  time.Time
}

type Result struct {
	Event *aping.Event
	Error error
}

// получить список рынков из cache
func (reader *reader) getFromCache(eventID int) (result *aping.Event) {
	reader.muCache.RLock()
	defer reader.muCache.RUnlock()
	if incache, f := reader.cache[eventID]; f && time.Since(incache.time) < time.Minute {
		result = &aping.Event{}
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

func getAPIResponse(eventID int) (markets []aping.Market, err error) {
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

	ep := aping.BettingAPI("listMarketCatalogue")

	responseBody, err := appkey.GetResponseWithAdminLogin(ep, &reqParams)
	if err != nil {
		return
	}
	var responseData struct {
		Result []aping.Market `json:"result"`
	}
	err = json.Unmarshal(responseBody, &responseData)
	if err != nil {
		err = fmt.Errorf("%q, %q", err, string(responseBody))
		return
	}

	for _, market := range responseData.Result {
		markets = append(markets, market)
	}

	if len(markets) == 0 {
		err = fmt.Errorf("no markets : %s", string(responseBody))
		return
	}

	return
}

func readMarketEvent(event *aping.Event) error {
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

	ep := aping.BettingAPI("listMarketCatalogue")

	responseBody, err := appkey.GetResponseWithAdminLogin(ep, &reqParams)
	if err != nil {
		return err
	}
	var xs struct {
		R []struct {
			EventType struct {
				ID   string `json:"id"`
				Name string `json:"name"`
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

	event.EventType = &aping.EventType{
		ID:   eventTypeID,
		Name: strings.Trim(x.EventType.Name, " ")}

	e := x.Event

	event.Name = strings.Trim(e.Name, " ")
	event.CountryCode = e.CountryCode
	event.Timezone = e.Timezone
	event.Venue = e.Venue
	event.OpenDate = e.OpenDate

	return nil
}

func readEvent(eventID int) (event *aping.Event, err error) {
	var markets []aping.Market

	markets, err = getAPIResponse(eventID)
	if err != nil {
		return
	}


	event = &aping.Event{ID: eventID}
	err = readMarketEvent(event)
	if err != nil {
		event = nil
		err = fmt.Errorf("%s, %d markets readed", err.Error(), len(markets))
		return
	}
	event.Markets = markets
	return
}
