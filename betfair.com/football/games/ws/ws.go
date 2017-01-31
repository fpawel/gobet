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
	log.Printf("begin ws football session: %v\n", conn.RemoteAddr())
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
	n := x.getConnIndex(conn)
	if n > -1 && n < len(x.openedSessions) {
		x.openedSessions = append(x.openedSessions[:n], x.openedSessions[n+1:]...)
	}
	x.mu.Unlock()
	conn.Close()
	log.Printf("end ws football session %v: %s\n", conn.RemoteAddr(), reason)
}

func (x *Handler) updateSession(session *session, games []football.Game, changes *update.Games) {

	err := session.WriteJSON(changes)
	websocketConn := session.websocketConn
	if err != nil {
		x.closeSession(websocketConn, fmt.Sprintf("write error: %v", err))
		return
	}
	time.Sleep(100 * time.Millisecond )
	messageType, recivedBytes, err := websocketConn.ReadMessage()
	if err != nil {
		log.Printf("read error %v: %v", websocketConn.RemoteAddr(), err)
		return
	}

	switch messageType {
	case websocket.CloseMessage:
		x.closeSession(websocketConn, "COLSE message recived from client")
	default:
		if string(recivedBytes) == changes.HashCode {
			x.mu.RLock()
			n := x.getConnIndex(websocketConn)
			x.mu.RUnlock()
			if n > -1 && n< len(x.openedSessions) {
				x.mu.Lock()
				x.openedSessions[n].games = games
				x.mu.Unlock()
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
				log.Printf("write error %v: %v", session.websocketConn.RemoteAddr(), err)
			}
			x.closeSession(session.websocketConn, fmt.Sprintf("ws football session games error: %v", err))
		}()

	}
}
