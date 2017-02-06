package ws

import (
	"github.com/gorilla/websocket"
	"github.com/user/gobet/betfair.com/football"
	"github.com/user/gobet/betfair.com/football/games/ws/session"
	"log"
	"sync"
)

type Handler struct {
	mu             *sync.RWMutex
	openedSessions []*session.Handle
}

func NewHandler() (x *Handler) {
	x = new(Handler)
	x.mu = new(sync.RWMutex)
	x.openedSessions = []*session.Handle{}
	return
}

func (x *Handler) getSession(hsession *session.Handle) (n int, ok bool) {
	n = -1
	for i, p := range x.openedSessions {
		if p == hsession {
			n = i
			ok = true
			break
		}
	}
	return
}

func (x *Handler) NewSession(conn *websocket.Conn) {

	session := session.Open(conn, func(hsession *session.Handle, reason error) {
		x.mu.Lock()
		n, ok := x.getSession(hsession)
		x.mu.Unlock()
		if !ok {
			log.Printf("closing  session not found conn=[%v], closing reason: %v\n",
				hsession.What(), reason)
			return
		}

		openedSessionsCount := len(x.openedSessions)
		x.openedSessions = append(x.openedSessions[:n], x.openedSessions[n+1:]...)

		log.Printf("end websocket session %d of %d conn=[%v]: %v\n",
			n, openedSessionsCount, hsession.What(), reason)
		log.Printf("%d opened sessions left\n", openedSessionsCount-1)
	})

	x.mu.Lock()
	x.openedSessions = append(x.openedSessions, session)
	openedSessionsCount := len(x.openedSessions)
	x.mu.Unlock()

	log.Printf("%d opened sessions\n", openedSessionsCount)

}

func (x *Handler) getOpenedSessions() (openedSessions []*session.Handle) {

	x.mu.RLock()
	for _, session := range x.openedSessions {
		openedSessions = append(openedSessions, session)
	}

	x.mu.RUnlock()
	return
}

func (x *Handler) NotifyNewGames(games []football.Game) {
	for _, hSession := range x.getOpenedSessions() {
		go hSession.Update(games)
	}
}

func (x *Handler) NotifyError(err error) {
	if err == nil {
		return
	}
	for _, hSession := range x.getOpenedSessions() {
		go hSession.NotifyInternalError(err)

	}
}
