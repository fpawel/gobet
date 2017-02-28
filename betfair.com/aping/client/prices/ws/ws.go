package ws

import (
	"github.com/gorilla/websocket"
	"log"
	"time"
	"github.com/user/gobet/betfair.com/aping/client/event"
	"github.com/user/gobet/betfair.com/aping/client"
)

type Session struct {
	conn *websocket.Conn
	eventID int
}

func NewSession(conn *websocket.Conn, eventID int) (session *Session) {
	session = &Session{conn : conn, eventID : eventID}
	return
}

func (session *Session) whatConn() string{
	return session.conn.RemoteAddr().String()
}

func (session *Session) run(){
	ch := make( chan event.Result )
	event.Get(session.eventID, ch)
	result := <- ch
	if result.Error != nil {
		session.internalError(result.Error)
		session.conn.Close()
		return
	}
	err := session.conn.WriteJSON( struct{
		Event *client.Event `json:"event"`
	} {result.Event })
	if err != nil {
		log.Printf("write error conn=%v: %v", session.whatConn(), err)
		session.conn.Close()
		return
	}



}

func (session *Session) internalError(internalError error) {

	err := session.conn.WriteJSON( struct{
		Error string `json:"error"`
	} {internalError.Error() })

	if err != nil {
		log.Printf("write error conn=%v: %v", session.whatConn(), err)
	}
	time.Sleep(10 * time.Second)
}

