package daemon

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi"
	"github.com/gorilla/websocket"
	"github.com/user/gobet/betfair.com/aping/client/event"
	"github.com/user/gobet/betfair.com/aping/client/eventPrices"
	"github.com/user/gobet/betfair.com/aping/client/eventPrices/eventPricesWS"
	"github.com/user/gobet/betfair.com/aping/client/eventTypes"
	"github.com/user/gobet/betfair.com/aping/client/events"

	"github.com/user/gobet/proxi"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"github.com/user/gobet/config"
	"github.com/user/gobet/data/footballMatches"
	"github.com/user/gobet/hub"

	"github.com/user/gobet/gate"
)

func Run() {
	router := chi.NewRouter()
	FileServer(router, "/", http.Dir("assets"))

	router.Get("/proxi/*", proxi.Proxi)

	var websocketUpgrader = websocket.Upgrader{} // use default options

	var hub = hub.New()
	var footballMatches = footballMatches.New(hub)
	footballMatches.Run()



	router.Get("/d", func(w http.ResponseWriter, r *http.Request) {
		client := gate.NewClient(w,r)
		hub.SubscribeFootball(client,true);
		client.Run(hub.UnregisterClient, func (recivedBytes []byte) {
			processDataFromPeer(hub, client, recivedBytes )
		})
	})
	router.Get("/football/games", func(w http.ResponseWriter, r *http.Request) {
		games, err := footballMatches.Get()
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
	setupRouteWebsocketPrices(&websocketUpgrader, router)
	http.ListenAndServe(":"+ config.Get().Port, router)

}

type Request struct {

	Football *struct{
		ConfirmHashCode string
	} `json:",omitempty"`

}

func  processDataFromPeer(hub *hub.Hub, c *gate.Client, bytes []byte) {
	var request Request
	if err := json.Unmarshal(bytes, &request); err != nil {
		c.SendJsonError("error demarshaling json request: " + err.Error())
		return
	}
	if request.Football != nil {
		hub.ConfirmFootball(c, request.Football.ConfirmHashCode)
	}
}

func setupRouteWebsocketPrices(websocketUpgrader *websocket.Upgrader, router chi.Router) {
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
	w.WriteHeader(http.StatusOK)
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
