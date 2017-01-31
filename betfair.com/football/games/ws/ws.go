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
	mu             *sync.RWMutex
	openedSessions []*session
}

type session struct {
	websocketConn *websocket.Conn
	muConn        *sync.Mutex

	games   []football.Game
	muGames *sync.RWMutex
}

func NewHandler() (x *Handler) {
	x = new(Handler)
	x.mu = new(sync.RWMutex)
	x.openedSessions = []*session{}
	return
}

func newSession(conn *websocket.Conn) (r *session) {
	log.Printf("begin ws football session conn=[%v]\n", conn.RemoteAddr())
	r = &session{}
	r.websocketConn = conn
	r.games = []football.Game{}
	r.muConn = new(sync.Mutex)
	r.muGames = new(sync.RWMutex)
	return
}

func (x *session) writeJSONSafely(i interface{}) error {
	x.muConn.Lock()
	defer x.muConn.Unlock()
	return x.websocketConn.WriteJSON(i)
}

func (x *session) readSafely() (messageType int, recivedBytes []byte, err error) {
	x.muConn.Lock()
	defer x.muConn.Unlock()
	messageType, recivedBytes, err = x.websocketConn.ReadMessage()
	return
}

func (x *session) update(changes *update.Games) (err error) {

	if changes == nil {
		return
	}

	err = x.writeJSONSafely(changes)

	if err != nil {
		return
	}
	time.Sleep(500 * time.Millisecond)

	messageType, recivedBytes, err := x.readSafely()

	if err != nil {
		time.Sleep(time.Second)
		return
	}

	switch messageType {
	case websocket.CloseMessage:
		err =  fmt.Errorf("%s", "client drope COLSE message")
		return
	default:
		recivedStr := string(recivedBytes)
		if recivedStr == changes.HashCode {
			return
		} else {
			time.Sleep(time.Second)

			err = fmt.Errorf("unexpected answer %v, expected %v",
				recivedStr, changes.HashCode)
			return
		}
	}
}

func (x *Handler) getSessionByConn(websocketConn *websocket.Conn) (n int, session *session) {
	n = -1
	for i, p := range x.openedSessions {
		if p.websocketConn == websocketConn {
			n = i
			session = p
			break
		}
	}
	return
}

func (x *Handler) NewSession(conn *websocket.Conn, games []football.Game) {

	session := newSession(conn)

	changes := update.New(nil, games)

	err := session.update(changes)
	if err == nil {
		session.games = games

		x.mu.Lock()
		x.openedSessions = append(x.openedSessions, session)
		openedSessionsCount := len(x.openedSessions)
		x.mu.Unlock()

		log.Printf("%d opened sessions\n", openedSessionsCount)

	} else {
		log.Printf("error opening websocket session %v: %v\n", conn.RemoteAddr(), err)
		conn.Close()
	}

}

func (x *Handler) closeSession(conn *websocket.Conn, reason error) {

	x.mu.Lock()
	sessionIndex, session := x.getSessionByConn(conn)
	if session == nil{
		conn.Close()
		x.mu.Unlock()
		log.Printf("session not found when closing %v: %v\n", reason, conn.RemoteAddr())

		return
	}

	openedSessionsCount := len(x.openedSessions)
	if sessionIndex > -1 && sessionIndex < openedSessionsCount {
		x.openedSessions = append(x.openedSessions[:sessionIndex], x.openedSessions[sessionIndex+1:]...)
	}
	conn.Close()
	x.mu.Unlock()

	log.Printf("end websocket session %d of %d conn=[%v]: %s\n",
		sessionIndex, openedSessionsCount, conn.RemoteAddr(), reason)
	log.Printf("%d opened sessions left\n", openedSessionsCount-1)
}

func (x *Handler) getOpenedSessions() (openedSessions []*session) {

	x.mu.RLock()
	for _,session := range x.openedSessions {
		openedSessions = append(openedSessions, session)
	}

	x.mu.RUnlock()
	return
}

func (x *Handler) updateSessionGames(session *session, games []football.Game) {
	session.muGames.RLock()
	changes := update.New(session.games, games)
	session.muGames.RUnlock()

	if changes != nil {
		err := session.update(changes)
		if err == nil {
			session.muGames.Lock()
			session.games = games
			session.muGames.Unlock()
		} else {
			x.closeSession(session.websocketConn, err)
		}
	}
}

func (x *Handler) NotifyNewGames(games []football.Game) {
	for _, session := range x.getOpenedSessions() {
		go x.updateSessionGames(session, games)
	}
}

func (x *Handler) NotifyError(err error) {
	if err == nil {
		return
	}

	errorInfo := &struct{ error string }{fmt.Sprintf("%v", err)}
	for _, session := range x.getOpenedSessions() {
		go func() {
			err := session.writeJSONSafely(errorInfo)
			if err != nil {
				log.Printf("write error conn=%v: %v", session.websocketConn.RemoteAddr(), err)
			}
			x.closeSession(session.websocketConn, fmt.Errorf("games error: %v", err))
		}()

	}
}
