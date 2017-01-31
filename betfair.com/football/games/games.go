package games

import (
	"github.com/gorilla/websocket"
	"github.com/user/gobet/betfair.com/football"
	"github.com/user/gobet/betfair.com/football/games/ws"
	"sync"
	"github.com/user/gobet/betfair.com/football/webclient"
	"os"
	"log"
	"time"
	"github.com/user/gobet/betfair.com/aping/client"
	"github.com/user/gobet/betfair.com/aping/client/events"
)


type listGames struct {
	mu    sync.RWMutex
	xs    []football.Game
	muErr sync.RWMutex
	err   error
	wsHandler 	  *ws.Handler
}

func New () (x *listGames){

	x = new(listGames)
	x.wsHandler = ws.NewHandler()
	go func() {
		for {
			x.update()
			if os.Getenv("MYMOBILEINET") == "true" {
				log.Println("MYMOBILEINET: sleep one minute")
				time.Sleep(time.Minute)
			}
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
	if e!=nil{
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
	games,err := x.Get()
	if err == nil {
		x.wsHandler.NotifyNewGames(games)
	} else{
		x.wsHandler.NotifyError(err)
	}
}

func (x *listGames) update() {

	mevents, err := getEvents()
	if err != nil {
		x.setError(err)
		return
	}

	firstPageURL, err := webclient.ReadFirstPageURL()
	if err != nil {
		x.setError(err)
		return
	}

	var readedGames []football.Game
	ptrNextPage := &firstPageURL
	for page := 0; ptrNextPage != nil && err == nil; page++ {

		var gamesPage []football.Game
		gamesPage, ptrNextPage, err = webclient.ReadPage(webclient.BetfairURL + *ptrNextPage)
		if err != nil {
			x.setError(err)
			return
		}

		for _, game := range gamesPage {
			game.Live.Page = page
			if event,ok := mevents[game.EventID] ; ok {
				game.Event = &event
				readedGames = append(readedGames, game)
			} else {
				// нет события Event с game.EventID.
				// Возможно, кеш событий "не свежий" и его следует обновить
				events.ClearCache(1)
			}
		}
	}
	if err == nil {
		x.setGames(readedGames)
	}
}

func getEvents() (mevents map[int]client.Event, err error) {

	ch := make(chan events.Result)
	events.Get(1, ch)
	r := <-ch
	err  = r.Error
	if r.Error != nil {
		return
	}

	mevents = make(map[int]client.Event)
	for _, event := range r.Events {
		mevents[event.ID] = event
	}
	return

}
