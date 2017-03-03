package eventPricesWS

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/user/gobet/betfair.com/aping/client"
	"github.com/user/gobet/betfair.com/aping/client/event"
	"github.com/user/gobet/betfair.com/aping/client/eventPrices"
	"github.com/user/gobet/utils"
	"hash/fnv"
	"log"
	"reflect"
	"strconv"
	"sync"
	"time"
)

const (
	BACK = "BACK"
	LAY = "LAY"
)

type Market struct {
	ID             string  `json:"id"`
	TotalMatched   float64 `json:"total_matched,omitempty"`
	TotalAvailable float64 `json:"total_available,omitempty"`
	Runners [] Runner `json:"runners,omitempty"`
}

type Runner struct {
	ID   int    `json:"id"`
	Odds []Odd  `json:"odds,omitempty"`
}

type Odd struct {
	Index  int  `json:"index"`
	Side  string  `json:"side"`
	Odd   * client.Odd`json:"odd"`
}


func oddDiff(n int, side string, x *client.Runner, y *client.Runner) (r *Odd){
	x_ := x.GetOdd(side,n)
	y_ := y.GetOdd(side,n)
	if !reflect.DeepEqual(x_,y_){
		r = &Odd{
			Index  : n,
			Side  : side,
			Odd   : y_,
		}
	}
	return
}

func getDiferenceOfMarkets(x *client.Market, y *client.Market, r *Market) (yes bool){
	r.ID = y.ID
	if x.TotalMatched != y.TotalMatched {
		yes = true
		r.TotalMatched = y.TotalMatched
	}

	if x.TotalAvailable != y.TotalAvailable {
		yes = true
		r.TotalAvailable = y.TotalAvailable
	}
	if len(x.Runners) != len(y.Runners){
		log.Println("ERROR getDiferenceOfMarkets:",x.Name,x.ID,
			"difernt numbers of runners", len(x.Runners), len(y.Runners))
		return
	}

	for nRunner,aX := range x.Runners{
		runner := Runner{ID:aX.ID}
		yesRunner := false
		aY := y.Runners[nRunner]
		for n := 0; n<3; n++{

			if back := oddDiff(n, BACK, &aX,&aY); back != nil {
				runner.Odds = append(runner.Odds, *back)
				yesRunner = true
			}
			if lay := oddDiff(n, LAY, &aX,&aY); lay != nil {
				runner.Odds = append(runner.Odds, *lay)
				yesRunner = true
			}
		}
		if yesRunner {
			yes = true
			r.Runners = append(r.Runners, runner)
		}
	}

	return
}


type Writer struct {
	conn        *websocket.Conn
	eventID     int
	id          string
	event       *client.Event
	marketIDs   map[string]interface{}
	muMarketIDs sync.RWMutex
}

func RegisterNewWriter(eventID int, conn *websocket.Conn) {
	r := &Writer{
		conn:        conn,
		eventID:     eventID,
		id:          utils.RandStringRunes(10),
		marketIDs:   make(map[string]interface{}),
		muMarketIDs: sync.RWMutex{},
		event:       nil,
	}
	openedSessions.addSession(r)
	conn.SetCloseHandler(nil)
	defaultCloseHandler := conn.CloseHandler()
	if defaultCloseHandler == nil {
		log.Panic("defaultcloseHandler is nil")
	}
	conn.SetCloseHandler(func(code int, text string) error {
		openedSessions.deleteSession(r.id)
		return defaultCloseHandler(code, text)
	})

	go r.run()

	return
}

func (session *Writer) run() {

	log.Println(session.what(), "wsprices started")
	defer log.Println(session.what(), "wsprices ended")

	// отправить потребителю идентификатор сессии
	isClosed := session.send(struct {
		SessionID string `json:"session_id"`
	}{SessionID: session.id})
	if isClosed {
		return
	}
	isClosed = session.sendInitEvent()
	for !isClosed {
		marketIDs := session.getMarketIDs()
		if len(marketIDs) == 0 {
			time.Sleep(2 * time.Second)
			continue
		}
		isClosed = session.processPrices(marketIDs)
	}
}

func (session *Writer) sendInitEvent() (isClosed bool) {

	ch := make(chan event.Result)
	event.Get(session.eventID, ch)
	result := <-ch

	if result.Error != nil {
		isClosed = true
		session.exitWithInternalError(
			"internal getting initial event", result.Error)
		return
	}
	session.event = result.Event

	// отправить потребителю объект result.Event
	isClosed = session.send(struct {
		Event *client.Event `json:"event"`
	}{Event: result.Event})

	return
}

func (session *Writer) processPrices(marketIDs []string) (isClosed bool) {
	ch := make(chan eventPrices.Result)

	eventPrices.Get(session.eventID, marketIDs, ch)
	result := <-ch

	if result.Error != nil {
		isClosed = true
		session.exitWithInternalError(
			"internal getting prices", result.Error)
		return
	}

	isClosed = session.setMarkets(result.Markets)
	return
}

func (session *Writer) setMarkets(markets []client.Market) (isClosed bool) {
	nmarkets := make(map[string]int)
	for n, m := range session.event.Markets {
		nmarkets[m.ID] = n
	}

	for _, nextMarket := range markets {
		nmarket, ok := nmarkets[nextMarket.ID]
		if !ok {
			log.Println("market ID not found", nextMarket.ID, session.what())
			continue
		}
		market := session.event.Markets[nmarket]

		diff := Market{}
		if getDiferenceOfMarkets(&market, &nextMarket, &diff){
			isClosed = session.send(struct {
				Market Market `json:"market"`
			}{Market: diff})
			if isClosed {
				return
			}
			session.event.Markets[nmarket] = nextMarket
		}


	}
	return
}

func (session *Writer) send(data interface{}) (isClosed bool) {
	var err error
	isClosed, err = session.send1(data)

	if isClosed || err != nil {
		return
	}

	if err != nil {
		log.Println( session.what(), "writing event error:", err.Error())
		session.conn.Close()
		isClosed = true
	}
	return
}

func (session *Writer) send1(data interface{}) (isClosed bool, err error) {

	var eventBytes []byte
	eventBytes, err = json.Marshal(data)
	if err != nil {
		err = session.newError("marhaling json", err)
		return
	}
	fnv32a := fnv.New64a()
	fnv32a.Write(eventBytes)

	dataToSend := &struct {
		Data     interface{} `json:"ok"`
		HashCode string      `json:"hash_code"`
	}{
		Data:     data,
		HashCode: strconv.FormatUint(fnv32a.Sum64(), 16),
	}
	err = session.conn.WriteJSON(dataToSend)
	if err != nil {
		err = session.newError("writing", err)
		return
	}
	messageType, recivedBytes, err := session.conn.ReadMessage()
	if err != nil {
		time.Sleep(time.Second)
		err = session.newError("reading", err)
		return
	}

	switch messageType {
	case websocket.CloseMessage:
		isClosed = true
		log.Printf("%s, client drope COLSE message\n",
			session.what())
		return
	default:
		if string(recivedBytes) == dataToSend.HashCode {
			return
		} else {
			err = fmt.Errorf("%s: unexpected answer [%s], expected hash code [%s]",
				session.what(),
				string(recivedBytes), dataToSend.HashCode)
			time.Sleep(time.Second)
			return
		}
	}
	return
}

func (session *Writer) internalError(internalError error) {
	err := session.conn.WriteJSON(struct {
		Error string `json:"error"`
	}{internalError.Error()})
	if err != nil {
		log.Println(session.what(), "failed writing internal error info", err.Error())
	}
}

func (session *Writer) newError(context string, err error) error {
	return fmt.Errorf(
		"%s: %s, %s",
		session.what(), context, err.Error())
}

func (session *Writer) exitWithInternalError(context string, err error) {
	session.internalError(session.newError(context, err))
	session.conn.Close()
	time.Sleep(10 * time.Second)
	return

}

func (session *Writer) what() string {
	s := ""
	if session.event != nil {
		s = fmt.Sprintf("-\"%s\"", session.event.Name)
	}
	return fmt.Sprintf("wsprices-writer-%d-%s-%s%s",
		session.eventID,
		session.conn.RemoteAddr().String(),
		session.id, s)
}

func (session *Writer) getMarketIDs() (marketIDs []string) {
	session.muMarketIDs.RLock()
	defer session.muMarketIDs.RUnlock()

	for marketID := range session.marketIDs {
		marketIDs = append(marketIDs, marketID)
	}
	return
}

func (session *Writer) setMarketID(ID string, include bool) {
	session.muMarketIDs.Lock()
	defer session.muMarketIDs.Unlock()
	if include {
		session.marketIDs[ID] = struct{}{}
	} else {
		delete(session.marketIDs, ID)
	}
}

func RegisterNewReader(ID string, conn *websocket.Conn) {

	what := fmt.Sprintf("wsprices-reader-%s",
		conn.RemoteAddr().String())

	trace := func(text string) {
		log.Println("wsprices-reader", what, ":", text)
	}
	newError := func(context string) {
		trace("error: " + context)
	}

	nestedError := func(context string, err error) {
		newError(context + ": " + err.Error())
	}

	writer, ok := openedSessions.getSession(ID)
	if !ok {
		conn.Close()
		newError("session ID not found")
		return
	}
	trace(fmt.Sprintf("new reader for %s", writer.what()))

	go func() {
		defer func() {
			conn.Close()
			log.Println("end of wsprices-reader", what)
		}()

		for {
			messageType, recivedBytes, err := conn.ReadMessage()
			if err != nil {
				time.Sleep(time.Second)
				nestedError("reading", err)
				return
			}

			switch messageType {
			case websocket.CloseMessage:
				trace("client drope COLSE message")
				return
			default:
				var x struct {
					MarketID string `json:"market_id"`
					Include  bool   `json:"include"`
				}
				if err := json.Unmarshal(recivedBytes, &x); err != nil {
					nestedError("unmarhaling", err)
					continue
				}

				writer, ok := openedSessions.getSession(ID)
				if !ok {
					trace("session was closed")
					return
				}
				writer.setMarketID(x.MarketID, x.Include)
			}
		}
	}()

	return
}

var openedSessions = func() (r struct {
	getSession    func(string) (*Writer, bool)
	addSession    func(*Writer)
	deleteSession func(string)
}) {
	xs := make(map[string]*Writer)
	mu := sync.RWMutex{}
	r.getSession = func(id string) (r *Writer, ok bool) {
		mu.RLock()
		defer mu.RUnlock()
		r, ok = xs[id]
		return
	}
	r.addSession = func(x *Writer) {
		mu.Lock()
		defer mu.Unlock()
		xs[x.id] = x
	}
	r.deleteSession = func(id string) {
		mu.Lock()
		defer mu.Unlock()
		delete(xs, id)
	}
	return
}()
