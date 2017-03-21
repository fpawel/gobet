package games

import (
	"sync"

	"github.com/gorilla/websocket"
	"github.com/user/gobet/betfair.com/football"
	"github.com/user/gobet/betfair.com/football/games/ws"
	"github.com/user/gobet/betfair.com/football/webclient"

	"github.com/user/gobet/betfair.com/aping/client"
	"github.com/user/gobet/betfair.com/aping/client/events"

	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

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

func getEvents() (mevents map[int]client.Event, err error) {

	ch := make(chan events.Result)
	events.Get(1, ch)
	r := <-ch
	err = r.Error
	if r.Error != nil {
		return
	}

	mevents = make(map[int]client.Event)
	for _, event := range r.Events {
		mevents[event.ID] = event
	}
	return

}

func readGames() (readedGames []football.Game, err error) {
	var firstPageURL string
	firstPageURL, err = webclient.ReadFirstPageURL()
	if err != nil {
		return
	}

	var mevents map[int]client.Event
	mevents, err = getEvents()
	if err != nil {
		return
	}

	ptrNextPage := &firstPageURL
	hasMissingEvents := false
	for page := 0; ptrNextPage != nil && err == nil; page++ {
		var gamesPage []football.Game
		gamesPage, ptrNextPage, err = webclient.ReadPage(webclient.BetfairURL + *ptrNextPage)
		if err != nil {
			return
		}
		for _, game := range gamesPage {
			game.Live.Page = page
			if event, ok := mevents[game.EventID]; ok {
				game.Event = &event
			} else {
				log.Println("footbal: missing event", game)
				// нет события Event с game.EventID.
				hasMissingEvents = true
			}
			readedGames = append(readedGames, game)
		}
	}

	if hasMissingEvents {
		// Возможно, кеш событий "не свежий" и его следует обновить
		events.ClearCache(1)
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
