package server

import (
	"encoding/json"
	"github.com/user/gobet/betfair.com/aping/client"
	"github.com/user/gobet/betfair.com/aping/client/eventTypes"
	"github.com/user/gobet/betfair.com/aping/client/events"
	"github.com/user/gobet/gate"
)

type Server struct {
	Hub      *Hub
	Football *Foootball
}

type Request struct {
	Football *struct {
		ConfirmHashCode string
	} `json:",omitempty"`

	ListEventTypes *struct{} `json:",omitempty"`

	ListEventType *int `json:",omitempty"`

	SubscribeFootball *bool `json:",omitempty"`
}

func NewServer() *Server {
	hub := NewHub()
	football := NewFoootball(hub)
	football.Run()
	return &Server{
		hub,
		football,
	}
}

func (x *Server) ProcessDataFromPeer(c *gate.Client, bytes []byte) {
	var request Request
	if err := json.Unmarshal(bytes, &request); err != nil {
		c.SendJsonError("error unmarshal json request: " + err.Error())
		return
	}

	if request.Football != nil {
		x.Hub.ConfirmFootball(c, request.Football.ConfirmHashCode)
		return
	}

	if request.SubscribeFootball != nil {
		x.Hub.SubscribeFootball(c, *request.SubscribeFootball)
		if *request.SubscribeFootball {
			footballMatches, err := x.Football.Get()
			if err == nil {
				x.Hub.FootballMatches <- footballMatches
			} else {
				x.Hub.FootballError <- err.Error()
			}
		}
		return
	}

	if request.ListEventTypes != nil {

		ch := make(chan eventTypes.Result)
		eventTypes.Get(ch)
		x := <-ch
		close(ch)

		if x.Error == nil {
			c.SendJson(struct{ EventTypes []client.EventType }{x.EventTypes})
		} else {
			c.SendJsonError(x.Error.Error())
		}
		return
	}

	if request.ListEventType != nil {

		ch := make(chan events.Result)
		events.Get(*request.ListEventType, ch)
		x := <-ch
		close(ch)

		if x.Error == nil {
			var r struct {
				EventType struct {
					ID     int
					Events []client.Event
				}
			}
			r.EventType.ID = *request.ListEventType
			r.EventType.Events = x.Events
			c.SendJson(&r)
		} else {
			c.SendJsonError(x.Error.Error())
		}
		return
	}
}
