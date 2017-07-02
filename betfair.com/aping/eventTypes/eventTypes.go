package eventTypes

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"sync"

	"github.com/user/gobet/betfair.com/aping"
	"github.com/user/gobet/betfair.com/aping/appkey"
)

type Result struct {
	EventTypes []aping.EventType
	Error      error
}

func Get(ch chan<- Result) {
	readerInstance.get(ch)
}

var readerInstance = newReader()

type reader struct {
	muAwaiters *sync.RWMutex
	awaiters   []chan<- Result
	muCache    *sync.RWMutex
	data       []aping.EventType
}

func newReader() (x *reader) {
	x = new(reader)
	x.muAwaiters = new(sync.RWMutex)
	x.awaiters = nil
	x.data = nil
	x.muCache = new(sync.RWMutex)
	return
}

func (reader *reader) get(ch chan<- Result) {

	// если список ожидающих каналов awaiters[eventID] не пуст,
	reader.muAwaiters.RLock()
	if len(reader.awaiters) > 0 {
		// в настоящий момент другой поток получает список событий в doReadEvents(eventTypeID).
		reader.muAwaiters.RUnlock()
		reader.muAwaiters.Lock()
		// добавить ещё один канал ch к списку awaiters[eventID] и вернуть управление
		reader.awaiters = append(reader.awaiters, ch)
		reader.muAwaiters.Unlock()
		return
	}
	reader.muAwaiters.RUnlock()

	// внести канал ch в список awaiters[eventID]
	reader.muAwaiters.Lock()
	reader.awaiters = []chan<- Result{ch}
	reader.muAwaiters.Unlock()

	go reader.performRead()

	return
}

func (reader *reader) performRead() {
	xs, err := getAPIResponse()
	log.Printf("event types: %v\n", err)

	reader.muAwaiters.Lock()
	awaiters := make([]chan<- Result, len(reader.awaiters))
	copy(awaiters, reader.awaiters)
	reader.awaiters = nil
	reader.muAwaiters.Unlock()
	for _, ch := range awaiters {
		ch <- Result{EventTypes: xs, Error: err}
	}
}

func getAPIResponse() (eventTypes []aping.EventType, err error) {

	var reqParams struct {
		Locale string   `json:"locale"`
		Filter struct{} `json:"filter"`
	}
	reqParams.Locale = "ru"
	ep := aping.BettingAPI("listEventTypes")
	var responseBody []byte
	responseBody, err = appkey.GetResponseWithAdminLogin(ep, &reqParams)
	if err != nil {
		return
	}
	var xs struct {
		R []struct {
			MarketCount int `json:"marketCount"`
			X           struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"eventType"`
		} `json:"result"`
	}
	err = json.Unmarshal(responseBody, &xs)
	if err != nil {
		err = fmt.Errorf("can`t get event types from %v: %v", string(responseBody), err)
		return
	}

	for _, x := range xs.R {
		var id int
		id, err = strconv.Atoi(x.X.ID)
		if err != nil {
			err = fmt.Errorf("wrong event type id %v: %v", x, err)
			return
		}
		y := aping.EventType{ID: id, Name: x.X.Name, MarketCount: x.MarketCount}
		eventTypes = append(eventTypes, y)
	}
	return
}
