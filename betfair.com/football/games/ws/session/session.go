package session

import (
	"github.com/gorilla/websocket"
	"github.com/user/gobet/betfair.com/football/games/update"
	"time"
	"fmt"
	"sync"
	"github.com/user/gobet/betfair.com/football"
	"log"
)


type Handle interface {
	Close(reason error)
	Update(games []football.Game) ()
	NotifyInternalError(err error)
	GetIsClosed() bool
	What() string
}

type handle struct {
	websocketConn *websocket.Conn
	mu         	  *sync.RWMutex
	games         []football.Game
	isClosed      bool
	onclose func(Handle, error)()
}

func Open(conn *websocket.Conn, onclose func(Handle, error)()) Handle {
	r := new(handle)
	r.websocketConn = conn
	r.games = []football.Game{}
	r.mu = new(sync.RWMutex)
	r.isClosed = false
	r.onclose = onclose
	log.Printf("begin ws football session conn=[%v]\n", r.What())

	return r
}

func (x *handle) What() string {
	return x.websocketConn.RemoteAddr().String()
}

func (x *handle) write(i interface{}) error {
	x.mu.Lock()
	defer x.mu.Unlock()
	return x.websocketConn.WriteJSON(i)
}

func (x *handle) read() (messageType int, recivedBytes []byte, err error) {
	x.mu.Lock()
	defer x.mu.Unlock()
	messageType, recivedBytes, err = x.websocketConn.ReadMessage()
	return
}

func (x *handle) NotifyInternalError(err error) {
	if err == nil {
		return
	}
	errorInfo := &struct{ error error } {err}
	err = x.write(errorInfo)
	if err != nil {
		log.Printf("write error conn=%v: %v", x.What(), err)
	}
	x.Close( err )
}

func (x *handle) Update(games []football.Game) {
	err := x.doUpdate(games)
	if err!=nil{
		x.Close(err)
	}
}

func (x *handle) doUpdate(games []football.Game) (err error) {

	x.mu.RLock()
	if x.isClosed{
		x.mu.RUnlock()
		return
	}
	changes := update.New(x.games, games)
	x.mu.RUnlock()

	if changes == nil {
		return
	}

	if x.GetIsClosed(){
		return
	}
	err = x.write(changes)
	if err != nil {
		return
	}
	time.Sleep(500 * time.Millisecond)


	if x.GetIsClosed(){
		return
	}
	messageType, recivedBytes, err := x.read()
	if err != nil {
		time.Sleep(time.Second)
		return
	}

	switch messageType {
	case websocket.CloseMessage:
		err = fmt.Errorf("%v: client drope COLSE message", x.What())
		return
	default:
		recivedStr := string(recivedBytes)
		if recivedStr == changes.HashCode {

			x.mu.Lock()
			x.games = games
			x.mu.Unlock()

			return
		} else {
			time.Sleep(time.Second)

			err = fmt.Errorf("unexpected answer %v, expected %v",
				recivedStr, changes.HashCode)
			return
		}
	}
}


func (x *handle) Close(reason error) {
	x.mu.Lock()
	x.websocketConn.Close()
	x.isClosed = true
	x.mu.Unlock()
	x.onclose(x,reason)
}

func (x *handle) GetIsClosed() bool{
	x.mu.RLock()
	defer x.mu.RUnlock()
	return x.isClosed
}