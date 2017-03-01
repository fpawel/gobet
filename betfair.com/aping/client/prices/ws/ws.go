package ws

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/user/gobet/betfair.com/aping/client"
	"github.com/user/gobet/betfair.com/aping/client/event"
	"hash/fnv"
	"log"
	"strconv"
	"time"
	"sync"
	"github.com/user/gobet/utils"
	"github.com/user/gobet/betfair.com/aping/client/prices"
	"reflect"
)

type Writer struct {
	conn    *websocket.Conn
	eventID int
	id string
	event *client.Event
	marketIDs map[int] interface{}
	muMarketIDs sync.RWMutex
}

func RegisterNewWriter(eventID int, conn *websocket.Conn)  {
	r := &Writer{
		conn:               conn,
		eventID           : eventID,
		id : utils.RandStringRunes(10),
		marketIDs : make(map[int] interface{}),
		muMarketIDs : sync.RWMutex{},
		event : nil,
	}
	openedSessions.addSession(r)
	conn.SetCloseHandler(nil)
	defaultCloseHandler := conn.CloseHandler()
	if defaultCloseHandler == nil {
		log.Panic("defaultcloseHandler is nil")
	}
	conn.SetCloseHandler( func(code int, text string) error{
		openedSessions.deleteSession(r.id)
		return defaultCloseHandler(code,text)
	})
	return
}

func (session *Writer) run() {

	// отправить потребителю идентификатор сессии
	isClosed := session.send( struct{
		SessionID string `json:"session_id"`
	}{SessionID: session.id})
	if isClosed {
		return
	}
	isClosed = session.sendInitEvent()
	for !isClosed  {
		marketIDs := session.getMarketIDs()
		if len(marketIDs)==0{
			time.Sleep(2 * time.Second)
			continue
		}
		isClosed = session.processPrices(marketIDs)
	}
}

func (session *Writer) sendInitEvent() (isClosed bool){

	ch := make(chan event.Result)
	event.Get(session.eventID, ch)
	result := <-ch

	if result.Error != nil {
		isClosed  = true
		session.exitWithInternalError(
				"internal getting initial event", result.Error)
		return
	}
	session.event = result.Event

	// отправить потребителю объект result.Event
	isClosed = session.send( struct{
		Event *client.Event `json:"event"`
	}{Event: result.Event})

	return
}

func (session *Writer) processPrices(marketIDs []string) (isClosed bool){
	ch := make(chan prices.Result)

	prices.Get(session.eventID, marketIDs, ch)
	result := <-ch

	if result.Error != nil {
		isClosed  = true
		session.exitWithInternalError(
			"internal getting initial event", result.Error)
		return
	}

	nmarkets := make (map[string]int)
	for n,m := range session.event.Markets{
		nmarkets[m.ID] = n
	}

	for _,nextMarket := range result.Markets{
		nmarket,ok := nmarkets[nextMarket.ID]
		if !ok {
			log.Println("market ID not found", nextMarket.ID, session.what())
			continue
		}
		market := session.event.Markets[nmarket]

		if !reflect.DeepEqual(market,nextMarket ){
			isClosed = session.send( struct{
				Market *client.Market `json:"market"`
			}{Market: &nextMarket})
			if isClosed {
				return
			}
			session.event.Markets[nmarket] = nextMarket
		}
	}
	return
}

func (session *Writer) send(data interface{}) (isClosed bool){
	var err error
	isClosed, err = session.send1( data)
	if isClosed || err != nil {
		if err != nil {
			err = session.newError("writing event", err)
			session.internalError(err)
		}
		session.conn.Close()
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
		Data    interface{} `json:"ok"`
		HashCode uint64        `json:"hash_code"`
	}{
		Data:    data,
		HashCode: fnv32a.Sum64(),
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
		var recivedValue uint64
		recivedValue, err = strconv.ParseUint(string(recivedBytes), 10, 64)
		if err != nil {
			err = session.newError("parsing hash code", err)
			return
		}
		if recivedValue == dataToSend.HashCode {
			return
		} else {
			err = fmt.Errorf("unexpected answer %x, expected hash code %x",
				recivedValue, dataToSend.HashCode)
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

func (session *Writer) exitWithInternalError(context string, err error){
	session.internalError( session.newError(context, err))
	session.conn.Close()
	time.Sleep(10 * time.Second)
	return

}

func (session *Writer) what() string {

	return fmt.Sprintf("wsprices-write-%d-%s",
		session.eventID, session.conn.RemoteAddr().String() )
}

func (session *Writer) getMarketIDs() (marketIDs [] string) {
	session.muMarketIDs.RLock()
	defer session.muMarketIDs.RUnlock()

	for marketID := range session.marketIDs {
		marketIDs = append(marketIDs, fmt.Sprintf("1.%d", marketID) )
	}
	return
}

type openedSessionsType struct{
	getSession func(string) (*Writer,bool)
	addSession func(*Writer)
	deleteSession func(string)
}

var openedSessions = func() openedSessionsType{

	xs := make(map[string] *Writer)
	mu := sync.RWMutex{}

	return  openedSessionsType{
		getSession : func (id string ) (r *Writer, ok bool) {
			mu.RLock()
			defer mu.RUnlock()
			r,ok = xs[id]
			return
		},
		addSession : func(x *Writer){
			mu.Lock()
			defer mu.Unlock()
			xs[x.id] = x
		},
		deleteSession :func(id string){
			mu.Lock()
			defer mu.Unlock()
			delete(xs,id)
		},
	}
}()


