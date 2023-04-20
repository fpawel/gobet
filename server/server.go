package server

import (
	"encoding/json"

	"gobet/betfair.com/aping"
	"gobet/betfair.com/aping/event"
	"gobet/betfair.com/aping/eventTypes"
	"gobet/betfair.com/aping/events"
	"gobet/gate"
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

	ListEvent *int `json:",omitempty"`

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
			c.SendJson(struct{ EventTypes []aping.EventType }{x.EventTypes})
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
					Events []aping.Event
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

	if request.ListEvent != nil {
		ch := make(chan event.Result)
		event.Get(*request.ListEvent, ch)
		x := <-ch
		close(ch)

		if x.Error == nil {
			var r struct {
				Event *aping.Event
			}
			r.Event = x.Event
			c.SendJson(&r)
		} else {
			c.SendJsonError(x.Error.Error())
		}
		return
	}
}
