package server

import (
	"github.com/user/gobet/betfair.com/aping/client/events"
	"github.com/user/gobet/betfair.com/football"
	"github.com/user/gobet/gate"
)

//Hub  maintains the set of active clients and broadcasts messages to the clients.
type Hub struct {
	unregister      chan *gate.Client // unregister requests from clients.
	update          chan clientStatePair
	FootballError   chan string
	FootballMatches chan []football.Match
	clients         map[*gate.Client]*clientState // Registered clients

}

type clientFootball struct {
	matches                    []football.Match // Football matches sent to the client
	hashCode                   string
	on                         bool
	waitConfirmationFromClient bool
}

type clientState struct {
	football clientFootball
}

type clientStatePair struct {
	client       *gate.Client
	stateHandler func(*clientState)
}

func NewHub() (h *Hub) {
	h = &Hub{
		update:          make(chan clientStatePair),
		unregister:      make(chan *gate.Client),
		clients:         make(map[*gate.Client]*clientState),
		FootballMatches: make(chan []football.Match),
		FootballError:   make(chan string),
	}
	go func() {
		for {
			h.work()
		}
	}()
	return
}

func (h *Hub) SubscribeFootball(c *gate.Client, on bool) {
	h.update <- clientStatePair{c, func(s *clientState) {
		s.football.on = on
		s.football.matches = nil
		s.football.hashCode = ""
		s.football.waitConfirmationFromClient = false
	}}
}

func (h *Hub) ConfirmFootball(c *gate.Client, confirmHashCode string) {
	h.update <- clientStatePair{c, func(s *clientState) {
		if s.football.on && s.football.hashCode == confirmHashCode {
			s.football.waitConfirmationFromClient = false
			s.football.hashCode = ""
		}
	}}
}

func (h *Hub) work() {
	select {

	case err := <-h.FootballError:
		for c, s := range h.clients {
			if s.football.on {
				c.SendJsonError(&footballError{
					err,
				})
			}
		}

	case newMatches := <-h.FootballMatches:
		var eventsResult events.Result
		for c, s := range h.clients {
			if s.football.on && !s.football.waitConfirmationFromClient {
				h.proccessFootball(c, s, newMatches, &eventsResult)
			}
		}

	case x := <-h.update:
		state, ok := h.clients[x.client]
		if !ok {
			state = &clientState{}
			h.clients[x.client] = state
		}
		x.stateHandler(state)

	case client := <-h.unregister:
		if _, ok := h.clients[client]; ok {
			delete(h.clients, client)
		}

	}
}

func (h *Hub) proccessFootball(c *gate.Client, s *clientState,
	newMatches []football.Match, eventsResult *events.Result) {
	changes := football.NewMatchesListChanges(s.football.matches, newMatches)
	if changes == nil {
		return
	}

	if len(changes.Inplay) > 0 {
		if len(eventsResult.Events) == 0 {
			ch := make(chan events.Result)
			events.Get(1, ch)
			*eventsResult = <-ch
		}

		if eventsResult.Error != nil {
			c.SendJsonError(&footballError{
				"events: " + eventsResult.Error.Error(),
			})
			return
		}
		changes.SetInplayEvents(eventsResult.Events)
	}

	var response struct {
		Football struct {
			Changes  *football.MatchesListChanges
			HashCode string
		}
	}
	response.Football.Changes = changes
	response.Football.HashCode = changes.GetHashCode()
	c.SendJson(response)
	s.football.matches = newMatches
	s.football.hashCode = response.Football.HashCode
	s.football.waitConfirmationFromClient = true
}

func (h *Hub) UnregisterClient(c *gate.Client) {
	h.unregister <- c
}

type footballError struct {
	FootballError string
}
