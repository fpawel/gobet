package daemon

import (
	"compress/gzip"
	"encoding/json"
	"fmt"

	"github.com/go-chi/chi"
	"github.com/gorilla/websocket"
	"github.com/user/gobet/betfair.com/aping/event"
	"github.com/user/gobet/betfair.com/aping/eventPrices"
	"github.com/user/gobet/betfair.com/aping/eventPrices/eventPricesWS"
	"github.com/user/gobet/betfair.com/aping/eventTypes"
	"github.com/user/gobet/betfair.com/aping/events"

	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"github.com/user/gobet/config"
	"github.com/user/gobet/server"
	"github.com/user/gobet/proxi"

	"github.com/user/gobet/gate"
)

func Run() {
	router := chi.NewRouter()
	FileServer(router, "/", http.Dir("assets"))

	router.Get("/proxi/*", proxi.Proxi)

	var server = server.NewServer()

	router.Get("/d", func(w http.ResponseWriter, r *http.Request) {
		client := gate.NewClient(w, r)
		//server.SubscribeFootball(client)
		client.Run(server.Hub.UnregisterClient, func(recivedBytes []byte) {
			server.ProcessDataFromPeer(client, recivedBytes)
		})
	})
	router.Get("/football/games", func(w http.ResponseWriter, r *http.Request) {
		games, err := server.Football.Get()
		jsonResult(w, games, err)
	})

	router.Get("/events/{eventTypeID}", func(w http.ResponseWriter, r *http.Request) {
		eventTypeID, err := strconv.Atoi(chi.URLParam(r, "eventTypeID"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		ch := make(chan events.Result)
		events.Get(eventTypeID, ch)
		x := <-ch
		close(ch)
		jsonResult(w, x.Events, x.Error)
	})

	setupRouterSports(router)
	setupRouteMarkets(router)
	setupRoutePrices(router)

	setupRouteWebsocketPrices(router)
	http.ListenAndServe(":"+config.Get().Port, router)

}

func setupRouteWebsocketPrices(router chi.Router) {
	var websocketUpgrader = websocket.Upgrader{} // use default options
	router.Get("/wsprices/{id}", func(w http.ResponseWriter, r *http.Request) {

		eventID, err := strconv.Atoi(chi.URLParam(r, "id"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		conn, err := websocketUpgrader.Upgrade(w, r, nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		conn.EnableWriteCompression(true)
		eventPricesWS.RegisterNewWriter(eventID, conn)

	})

	router.Post("/prices-markets", func(w http.ResponseWriter, r *http.Request) {
		requestBody, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		err = eventPricesWS.SetIncludeMarket(requestBody)
		jsonResult(w, struct{}{}, err)
	})
}

func setupRouterSports(router chi.Router) {
	router.Get("/sports", func(w http.ResponseWriter, r *http.Request) {
		ch := make(chan eventTypes.Result)
		eventTypes.Get(ch)
		x := <-ch
		close(ch)
		jsonResult(w, x.EventTypes, x.Error)
	})
}

func setupRouteMarkets(router chi.Router) {

	router.Get("/event/{id}", func(w http.ResponseWriter, r *http.Request) {

		eventID, err := strconv.Atoi(chi.URLParam(r, "id"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		ch := make(chan event.Result)
		event.Get(eventID, ch)
		x := <-ch
		close(ch)
		jsonResult(w, x.Event, x.Error)
	})
}

func setupRoutePrices(router chi.Router) {

	router.Get("/prices/{eventID}/{marketsIDs}", func(w http.ResponseWriter, r *http.Request) {

		strEventID := chi.URLParam(r, "eventID")
		eventID, err := strconv.Atoi(strEventID)
		if err != nil {
			http.Error(w, fmt.Sprintf("eventID: %v, %v", strEventID, err.Error()), http.StatusBadRequest)
			return
		}

		marketIDs := strings.Split(strings.Trim(chi.URLParam(r, "marketsIDs"), "/ "), "/")

		if len(marketIDs) == 0 {
			http.Error(w, fmt.Sprintf("%s, no markets requested", err.Error()), http.StatusBadRequest)
			return
		}

		ch := make(chan eventPrices.Result)
		eventPrices.Get(eventID, marketIDs, ch)
		x := <-ch
		close(ch)
		jsonResult(w, x.Markets, x.Error)
	})
}

func jsonResult(w http.ResponseWriter, data interface{}, err error) {

	if err != nil {
		var y struct {
			Error string `json:"error"`
		}
		y.Error = err.Error()
		setCompressedJSON(w, &y)
		return
	}

	var y struct {
		Ok interface{} `json:"ok"`
	}
	y.Ok = data
	setCompressedJSON(w, &y)

}

func setCompressedJSON(w http.ResponseWriter, data interface{}) {
	gz, err := gzip.NewWriterLevel(w, gzip.DefaultCompression)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer gz.Close()

	w.Header().Set("Content-Encoding", "gzip")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	encoder := json.NewEncoder(gz)
	encoder.SetIndent("", "    ")

	if err = encoder.Encode(data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// FileServer conveniently sets up a http.FileServer handler to serve
// static files from a http.FileSystem.
func FileServer(r chi.Router, path string, root http.FileSystem) {
	if strings.ContainsAny(path, "{}*") {
		panic("FileServer does not permit URL parameters.")
	}

	fs := http.StripPrefix(path, http.FileServer(root))

	if path != "/" && path[len(path)-1] != '/' {
		r.Get(path, http.RedirectHandler(path+"/", 301).ServeHTTP)
		path += "/"
	}
	path += "*"

	r.Get(path, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fs.ServeHTTP(w, r)
	}))
}
