package footballGames

import (
	"sync"

	"github.com/gorilla/websocket"
	"github.com/user/gobet/betfair.com/football"
	"github.com/user/gobet/betfair.com/football/footballGames/ws"
	"github.com/user/gobet/betfair.com/football/webclient"

	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
	"github.com/user/gobet/traficControl"
	"github.com/user/gobet/config"
)

type listGames struct {
	mu        sync.RWMutex
	games     []football.Game
	muErr     sync.RWMutex
	err       error
	wsHandler *ws.Handler
}



func New() (x *listGames) {
	x = &listGames{}
	x.wsHandler = ws.NewHandler()
	go func() {
		for {
			x.updateGames()
		}
	}()
	return
}

func (x *listGames) setGames(ys []football.Game) {
	x.mu.Lock()
	x.games = ys
	x.mu.Unlock()
	x.setError(nil)
	x.wsHandler.NotifyNewGames(ys)
}

func (x *listGames) setError(e error) {
	x.muErr.Lock()
	x.err = e
	x.muErr.Unlock()
	if e != nil {
		x.wsHandler.NotifyError(e)
	}
}

func (x *listGames) Get() (r []football.Game, err error) {
	x.muErr.RLock()
	defer x.muErr.RUnlock()

	if x.err != nil {
		err = x.err
		return
	}

	x.mu.RLock()
	defer x.mu.RUnlock()
	for _, game := range x.games {
		r = append(r, game)
	}
	return
}

func (x *listGames) OpenWebSocketSession(conn *websocket.Conn) {

	x.wsHandler.NewSession(conn)
	games, err := x.Get()
	if err == nil {
		x.wsHandler.NotifyNewGames(games)
	} else {
		x.wsHandler.NotifyError(err)
	}
}

func (x *listGames) updateGames() {

	if !config.ConstantlyUpdate {
		x.mu.RLock()
		gamesCount := len(x.games)
		x.mu.RUnlock()
		if gamesCount > 0 {
			return
		}
	}
	var nextGames []football.Game
	var err error
	if config.ReadFromHerokuApp {
		nextGames, err = getGamesListFromHerokuApp()
		time.Sleep( 20 * time.Second)
	} else {
		nextGames, err = readGames()
	}
	if err != nil {
		x.setError(err)
		return
	}
	x.setGames(nextGames)

}

func readGames() (readedGames []football.Game, err error) {
	var firstPageURL string
	firstPageURL, err = webclient.ReadFirstPageURL()
	if err != nil {
		return
	}

	ptrNextPage := &firstPageURL
	for page := 0; ptrNextPage != nil && err == nil; page++ {
		var gamesPage []football.Game
		gamesPage, ptrNextPage, err = webclient.ReadPage(webclient.BetfairURL + *ptrNextPage)
		if err != nil {
			return
		}
		for _, game := range gamesPage {
			game.Page = page
			readedGames = append(readedGames, game)
		}
	}

	return
}

func getGamesListFromHerokuApp() (readedGames []football.Game, err error) {

	var resp *http.Response
	url := "http://gobet.herokuapp.com/football/footballGames"
	resp, err = http.Get(url)
	if err != nil {
		err = fmt.Errorf("http error of %v: %v", url, err)
		return
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	traficControl.AddTotalBytesReaded(len(body), "HEROKU APP")

	var data struct {
		Ok  []football.Game `json:"ok,omitempty"`
		Err error           `json:"error,omitempty"`
	}

	err = json.Unmarshal(body, &data)
	if err != nil {
		err = fmt.Errorf("data error of %v: %v", url, err)
		return
	}
	readedGames = data.Ok
	return
}
