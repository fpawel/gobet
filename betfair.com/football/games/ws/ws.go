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
	mu *sync.RWMutex
	openedSessions []session
}

type session struct {
	websocketConn *websocket.Conn
	muConn	*sync.Mutex

	games   []football.Game
	muGames	*sync.RWMutex

}

func NewHandler() (x *Handler){
	x = new (Handler)
	x.mu = new (sync.RWMutex)
	x.openedSessions = []session{}
	return
}

func (x *session) writeJSONSafely(i interface{}) error{
	x.muConn.Lock()
	defer x.muConn.Unlock()
	return  x.websocketConn.WriteJSON(i)
}

func (x *session) readSafely() (messageType int, recivedBytes []byte, err error) {
	x.muConn.Lock()
	defer x.muConn.Unlock()
	messageType, recivedBytes, err = x.websocketConn.ReadMessage()
	return
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

func (x *Handler) NewSession(conn *websocket.Conn, games []football.Game) {
	log.Printf("begin ws football session conn=[%v]\n", conn.RemoteAddr())
	session := session{}
	session.websocketConn = conn
	session.games = games
	session.muConn = new(sync.Mutex)
	session.muGames = new(sync.RWMutex)

	x.mu.Lock()
	x.openedSessions = append(x.openedSessions,session )
	openedSessionsCount := len (x.openedSessions)
	x.mu.Unlock()
	log.Printf("%d opened sessions\n", openedSessionsCount)

}

func (x *Handler) closeSession(conn *websocket.Conn, reason error) {

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

func (x *Handler) updateSession(session *session, changes *update.Games) (err error) {


	err = session.writeJSONSafely(changes)

	if err != nil {
		return
	}
	time.Sleep(500 * time.Millisecond )

	messageType, recivedBytes, err := session.readSafely()

	if err != nil {
		time.Sleep(time.Second)
		return
	}

	switch messageType {
	case websocket.CloseMessage:
		return fmt.Errorf("%s", "client drope COLSE message")
	default:
		recivedStr := string(recivedBytes)
		if recivedStr == changes.HashCode {
			return
		} else{
			time.Sleep(time.Second)

				fmt.Errorf("unexpected answer %v, expected %v",
					recivedStr, changes.HashCode  )
			return
		}
	}
}

func (x *Handler) getOpenedSessions()(openedSessions []session) {

	x.mu.RLock()
	copy(openedSessions, x.openedSessions)
	x.mu.RUnlock()
	return
}

func (x *Handler) NotifyNewGames(games []football.Game) {

	for _, session := range x.getOpenedSessions() {
		go func() {
			session.muGames.RLock()
			sessionGames := session.games
			session.muGames.RUnlock()

			changes := update.New(sessionGames, games)

			if changes != nil {

				err := x.updateSession(&session, changes)
				if err == nil {
					session.muGames.Lock()
					session.games = games
					session.muGames.Unlock()
				} else {
					x.closeSession(session.websocketConn, err )
				}
			}
		}()

	}
}

func (x *Handler) NotifyError(err error) {
	if err == nil {
		return
	}

	errorInfo := &struct {error  string} { fmt.Sprintf("%v", err) }
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
