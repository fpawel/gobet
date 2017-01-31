package ws

import (
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/user/gobet/betfair.com/football"
	"github.com/user/gobet/betfair.com/football/games/update"
	"log"
	"sync"
	"time"
)

type Handler struct {
	mu sync.RWMutex
	openedSessions []session
}

type session struct {
	websocketConn *websocket.Conn
	games         []football.Game
	mu	*sync.Mutex
}

func (x *session) WriteJSON (i interface{}) error{
	x.mu.Lock()
	defer x.mu.Unlock()
	return  x.websocketConn.WriteJSON(i)
}

func (x *Handler) getConnIndex(websocketConn *websocket.Conn) (n int) {
	n = -1
	for i, item := range x.openedSessions {
		if item.websocketConn == websocketConn {
			n = i
			break
		}
	}
	return
}

func (x *Handler) OpenSession(conn *websocket.Conn, games []football.Game) {
	log.Printf("begin ws football session conn=[%v]\n", conn.RemoteAddr())
	session := session{}
	session.websocketConn = conn
	session.games = games
	session.mu = &sync.Mutex{}

	x.mu.Lock()
	x.openedSessions = append(x.openedSessions,session )
	x.mu.Unlock()

}

func (x *Handler) closeSession(conn *websocket.Conn, reason string) {

	x.mu.Lock()
	sessionIndex := x.getConnIndex(conn)
	openedSessionsCount := len(x.openedSessions)
	if sessionIndex > -1 && sessionIndex < openedSessionsCount {
		x.openedSessions = append(x.openedSessions[:sessionIndex], x.openedSessions[sessionIndex+1:]...)
	}
	x.mu.Unlock()
	conn.Close()
	log.Printf("end websocket session %d of %d conn=[%v]: %s\n",
		sessionIndex, openedSessionsCount, conn.RemoteAddr(), reason)
	log.Printf("%d opened sessions left\n", openedSessionsCount - 1)
}

func (x *Handler) updateSession(session *session, games []football.Game, changes *update.Games) {

	errWrite := session.WriteJSON(changes)
	websocketConn := session.websocketConn
	if errWrite != nil {
		x.closeSession(websocketConn, fmt.Sprintf("write websocket error: %v", errWrite))
		return
	}
	time.Sleep(500 * time.Millisecond )


	messageType, recivedBytes, errRead := websocketConn.ReadMessage()
	switch messageType {
	case websocket.CloseMessage:
		x.closeSession(websocketConn, "COLSE message recived from client")
	default:
		recivedStr := string(recivedBytes)
		if recivedStr == changes.HashCode {
			x.mu.RLock()
			n := x.getConnIndex(websocketConn)
			x.mu.RUnlock()
			if n > -1 && n< len(x.openedSessions) {
				x.mu.Lock()
				x.openedSessions[n].games = games
				x.mu.Unlock()
			}
		} else {
			if errRead != nil {
				log.Printf("read websocket error message_type=%d message=%v conn=[%v]: %v",
					messageType, recivedStr, websocketConn.RemoteAddr(), errRead)
				return
			}
		}
	}
	return
}

func (x *Handler) getOpenedSessions()(openedSessions []*session) {

	x.mu.RLock()
	for n := range x.openedSessions {
		openedSessions = append(openedSessions, &x.openedSessions[n])
	}
	x.mu.RUnlock()
	return
}

func (x *Handler) NotifyNewGames(games []football.Game) {

	for _, pSession := range x.getOpenedSessions() {
		changes := update.New(pSession.games, games)
		if changes != nil {
			go x.updateSession(pSession, games, changes)
		}
	}
}

func (x *Handler) NotifyError(err error) {
	if err == nil {
		return
	}

	errorInfo := &struct {error  string} { fmt.Sprintf("%v", err) }
	for _, session := range x.getOpenedSessions() {
		go func() {
			err := session.WriteJSON(errorInfo)
			if err != nil {
				log.Printf("write error conn=%v: %v", session.websocketConn.RemoteAddr(), err)
			}
			x.closeSession(session.websocketConn, fmt.Sprintf("ws football session games error: %v", err))
		}()

	}
}
