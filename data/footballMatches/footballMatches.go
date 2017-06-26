package footballMatches

import (
	"sync"

	"github.com/user/gobet/betfair.com/football"
	"github.com/user/gobet/betfair.com/football/webclient"

	"github.com/user/gobet/config"
	"github.com/user/gobet/hub"
	"time"
)

type L struct {
	mu      sync.RWMutex
	matches []football.Match
	err     error
	hub     *hub.Hub
}

func New(hub *hub.Hub) (x *L) {
	x = &L{
		hub : hub,
	}
	return
}

func (x *L) Run()  {
	go func() {
		for {
			x.updateGames()
		}
	}()
}




func (x *L) Get() (r []football.Match, err error) {
	x.mu.RLock()
	defer x.mu.RUnlock()

	if x.err != nil {
		err = x.err
		return
	}

	r = append(r, x.matches...)
	return
}

func (x *L) updateGames() {

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


