package games

import (
	"sync"

	"github.com/gorilla/websocket"
	"github.com/user/gobet/betfair.com/football"
	"github.com/user/gobet/betfair.com/football/games/ws"
	"github.com/user/gobet/betfair.com/football/webclient"


	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/user/gobet/envvars"
	"github.com/user/gobet/mobileinet"
)

type listGames struct {
	mu        sync.RWMutex
	xs        []football.Game
	muErr     sync.RWMutex
	err       error
	wsHandler *ws.Handler
}

func New() (x *listGames) {

	x = new(listGames)
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
	x.xs = ys
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
	for _, game := range x.xs {
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

	if !envvars.RunFootbal() {
		x.mu.RLock()
		hasGames := len(x.xs) > 0
		x.mu.RUnlock()
		if hasGames {
			return
		}
	}

	sleepTime := time.Second
	readGamesfunc := readGames
	if os.Getenv("MYMOBILEINET") == "true" {
		readGamesfunc = getGamesListFromHerokuApp
		sleepTime = 20 * time.Second
	}
	nextGames, err := readGamesfunc()
	if err != nil {
		x.setError(err)
		return
	}
	x.setGames(nextGames)
	time.Sleep(sleepTime)
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
	url := "http://gobet.herokuapp.com/football/games"
	resp, err = http.Get(url)
	if err != nil {
		err = fmt.Errorf("http error of %v: %v", url, err)
		return
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	mobileinet.LogAddTotalBytesReaded(len(body), "HEROKU APP")

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
