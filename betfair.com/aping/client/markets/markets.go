package markets

import (
	"encoding/json"
	"github.com/user/gobet/betfair.com/aping/client"
	"github.com/user/gobet/betfair.com/aping/client/endpoint"
	"github.com/user/gobet/betfair.com/aping/client/appkey"
	"strconv"
	"sync"
	"time"
	"log"
	"fmt"
	"strings"
	"errors"
)

func Get(eventID int, needRunners bool, ch chan<- Result){
	if needRunners {
		runnerReader.get(eventID,ch)
		return
	}
	xs := runnerReader.getFromCache(eventID)
	if xs != nil{
		r := Result{xs,nil}
		go func(){
			ch <- r.copy(false)
		}()
		return
	}
	withoutRunnerReader.get(eventID,ch)
}

var runnerReader = newReader(true)
var withoutRunnerReader = newReader(false)

type reader struct{
	muAwaiters *sync.RWMutex
	awaiters map[int][]chan<- Result
	muCache *sync.RWMutex
	cache map[int]listMarketsTime
	needRunners bool

}

func newReader(needRunners bool) (x *reader){
	x = new (reader)
	x.muAwaiters = new(sync.RWMutex)
	x.awaiters = make(map[int][]chan<- Result)
	x.cache = make(map[int]listMarketsTime)
	x.muCache = new(sync.RWMutex)
	x.needRunners = needRunners
	return
}

type listMarketsTime struct{
	markets []client.Market
	time    time.Time
}


type Result struct {
	Markets []client.Market `json:"result,omitempty"`
	Error   error  `json:"error,omitempty"`
}

func (x *Result) copy(needRunners bool) (y Result){
	y.Error = x.Error
	for _,m := range x.Markets {
		var m_ client.Market
		m_ = m
		if !needRunners {
			m_.Runners = nil
		}

		y.Markets = append(y.Markets, m_)
	}

	return
}

// получить список рынков из cache
func (reader *reader) getFromCache(eventID int) (markets []client.Market){
	reader.muCache.RLock()
	defer reader.muCache.RUnlock()
	if eventData, f := reader.cache[eventID]; f && time.Since(eventData.time) <  time.Minute  {
		markets = append(eventData.markets[:])
	}
	return
}

func (reader *reader) doDirectRead(eventID int) {
	result := Result{}
	result.Markets, result.Error = reader.directRead(eventID)
	log.Printf("read markets %d %v: %v\n", eventID, reader.needRunners, result.Error)

	if result.Error == nil {
		x := listMarketsTime{}
		x.time = time.Now()
		x.markets = append(result.Markets)
		reader.muCache.Lock()
		reader.cache[eventID] = x
		reader.muCache.Unlock()
	}

	reader.muAwaiters.RLock()
	thisAwaiters, ok := reader.awaiters[eventID]
	if !ok || len(thisAwaiters) == 0 {
		log.Fatalln("awaiters list is empty")
	}
	reader.muAwaiters.RUnlock()

	for _, ch := range thisAwaiters {
		ch <- result.copy(reader.needRunners)
	}

	reader.muAwaiters.Lock()
	delete(reader.awaiters, eventID)
	reader.muAwaiters.Unlock()

}

func (reader *reader) clear(eventID int){
	reader.muCache.Lock()
	defer reader.muCache.Unlock()
	delete(reader.cache, eventID)
}

func (reader *reader) get(eventID int, ch chan<- Result) {

	// если удалось получить список рынков из cache, записать список рынков в канал
	if cacheMarkets := reader.getFromCache(eventID); cacheMarkets != nil  {
		go func() {
			ch <- Result{cacheMarkets,nil}
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

	go reader.doDirectRead(eventID)

	return
}

func extractMarketID(s string) (id int, err error){
	xs := strings.Split(s, "1.")
	if len(xs) == 2 {
		id,err = strconv.Atoi(xs[1])
	} else{
		err = fmt.Errorf("%q is not valid market id", s)
	}
	return

}

func (reader *reader) directRead(eventID int) (markets []client.Market, err error) {
	var reqParams struct {
		Locale string `json:"locale"`
		MarketProjection []string `json:"marketProjection"`
		Filter struct {
			EventIDs []int `json:"eventIds"`
		} `json:"filter"`
		MaxResults int `json:"maxResults"`
	}

	reqParams.Locale = "ru"
	reqParams.Filter.EventIDs = []int{eventID}

	if reader.needRunners {
		reqParams.MarketProjection = []string{ "RUNNER_DESCRIPTION" }
	} else {
		reqParams.MarketProjection = []string{ }
	}
	reqParams.MaxResults = 1000


	ep := endpoint.BettingAPI("listMarketCatalogue")
	responseBody, err := appkey.GetResponse(ep, &reqParams)
	if err != nil {
		return
	}
	var xs struct {
		 R []struct {
			 client.MarketBase
			 ID string `json:"marketId"`
		 } `json:"result"`
	}
	err = json.Unmarshal(responseBody, &xs)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("%q, %q", err, string(responseBody)))
	}
	for _, x := range xs.R {
		var market client.Market
		market.MarketBase = x.MarketBase
		market.ID, err = extractMarketID(x.ID)
		if err != nil {
			return
		}

		markets = append(markets, market)
	}
	return
}



