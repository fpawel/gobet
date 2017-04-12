package eventPrices

import (
	"encoding/json"
	"fmt"
	"github.com/user/gobet/betfair.com/aping/client"
	"github.com/user/gobet/betfair.com/aping/client/appkey"
	"github.com/user/gobet/betfair.com/aping/client/endpoint"
	"log"
	"sync"
	"time"
)

type reader struct {
	muAwaiters *sync.RWMutex
	awaiters   map[int][]chan<- Result
	muCache    *sync.RWMutex
	cache      map[int]cacheItemType
}

type Result struct {
	Markets []client.Market
	Error   error
}

type cacheItemType struct {
	markets []client.Market
	time    time.Time
}

func Get(eventID int, marketIDs []string, ch chan<- Result) {
	readerInstance.get(eventID, marketIDs, ch)
}

var readerInstance = newReader()

func newReader() (x *reader) {
	x = new(reader)
	x.muAwaiters = new(sync.RWMutex)
	x.awaiters = make(map[int][]chan<- Result)
	x.cache = make(map[int]cacheItemType)
	x.muCache = new(sync.RWMutex)
	return
}

func (reader *reader) getFromCache(eventID int) (result []client.Market) {
	reader.muCache.RLock()
	defer reader.muCache.RUnlock()
	if incache, ok := reader.cache[eventID]; ok && time.Since(incache.time) < 2*time.Second {
		result = incache.markets
	}
	return
}

func (reader *reader) performReadMarkets(eventID int, marketIDs []string) {

	markets, err := readMarkets(eventID, marketIDs)
	log.Printf("read markets %d: %v\n", eventID, err)

	if err == nil {
		x := cacheItemType{markets, time.Now()}
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
		ch <- Result{markets, err}
	}
}

func (reader *reader) get(eventID int, marketIDs []string, ch chan<- Result) {

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

	go reader.performReadMarkets(eventID, marketIDs)

	return
}

func readMarkets(eventID int, marketIDs []string) (markets []client.Market, err error) {

	var reqParams struct {
		Locale          string   `json:"locale"`
		Markets         []string `json:"marketIds"`
		PriceProjection struct {
			PriceData  []string `json:"priceData"`
			Virtualise bool     `json:"virtualise"`
		} `json:"priceProjection"`
	}

	reqParams.Locale = "ru"
	reqParams.Markets = marketIDs
	reqParams.PriceProjection.PriceData = []string{"EX_BEST_OFFERS"}
	reqParams.PriceProjection.Virtualise = true

	ep := endpoint.BettingAPI("listMarketBook")

	responseBody, err := appkey.GetResponseWithAdminLogin(ep, &reqParams)
	if err != nil {
		return
	}
	var responseData struct {
		Result []client.Market `json:"result"`
	}
	err = json.Unmarshal(responseBody, &responseData)
	if err != nil {
		err = fmt.Errorf("%q, %q", err, string(responseBody))
		return
	}

	markets = responseData.Result

	return
}
