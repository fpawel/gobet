package server

import (
	"sync"

	"gobet/betfair.com/football"
	"gobet/betfair.com/football/webclient"

	"gobet/config"
	"time"
)

type Foootball struct {
	mu      sync.RWMutex
	matches []football.Match
	err     error
	hub     *Hub
}

func NewFoootball(hub *Hub) (x *Foootball) {
	x = &Foootball{
		hub: hub,
	}
	return
}

func (x *Foootball) Run() {
	go func() {
		for {
			x.updateGames()
		}
	}()
}

func (x *Foootball) Get() (r []football.Match, err error) {
	x.mu.RLock()
	defer x.mu.RUnlock()

	if x.err != nil {
		err = x.err
		return
	}

	r = append(r, x.matches...)
	return
}

func (x *Foootball) updateGames() {

	if !config.Get().ConstantlyUpdate {
		x.mu.RLock()
		gamesCount := len(x.matches)
		x.mu.RUnlock()
		if gamesCount > 0 {
			return
		}
	}

	var nextGames []football.Match
	var err error

	if config.Get().ReadFromHerokuApp {
		nextGames, err = webclient.ReadMatchesFromHerokuApp()
		time.Sleep(20 * time.Second)
	} else {
		nextGames, err = webclient.ReadMatches()
	}
	x.mu.Lock()

	if err != nil {
		x.err = err
		x.hub.FootballError <- err.Error()
	} else {
		x.err = nil
		x.matches = nextGames
		x.hub.FootballMatches <- nextGames
	}
	x.mu.Unlock()

}
