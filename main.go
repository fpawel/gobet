package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"

	"compress/gzip"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"github.com/user/gobet/betfair.com/football/games"
	"github.com/user/gobet/proxi"
	"github.com/user/gobet/utils"

	"strconv"

	"github.com/user/gobet/betfair.com/aping/client/event"
	"github.com/user/gobet/betfair.com/aping/client/eventTypes"
	"github.com/user/gobet/betfair.com/aping/client/events"

	"github.com/user/gobet/betfair.com/aping/client/eventPrices"
	"github.com/user/gobet/betfair.com/aping/client/eventPrices/eventPricesWS"
	"github.com/user/gobet/envvars"
)

var footbalGames = games.New()
var websocketUpgrader = websocket.Upgrader{} // use default options

func main() {
	setupRouter()
}

func setupRouter() {
	router := gin.Default()
	//router.Use(gzip.Gzip(gzip.DefaultCompression))
	//router.Use(gin.Logger())
	router.GET("proxi/*url", proxi.Proxi)
	router.GET("football", func(c *gin.Context) {
		conn, err := websocketUpgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			returnInternalServerError(c, err)
			return
		}
		conn.EnableWriteCompression(true)
		footbalGames.OpenWebSocketSession(conn)

	})
	router.GET("football/games", func(c *gin.Context) {
		games, err := footbalGames.Get()
		jsonResult(c, games, err)
	})
	setupRouterSports(router)
	setupRouterEvents(router)
	setupRouteMarkets(router)
	setupRoutePrices(router)
	setupRouteWebsocketPrices(router)
	setupRouteStatic(router)
	router.Run(":" + envvars.Port())
}

func setupRouteWebsocketPrices(router *gin.Engine) {
	router.GET("wsprices/:ID", func(c *gin.Context) {

		eventID, err := strconv.Atoi(c.Param("ID"))
		if err != nil {
			c.String(http.StatusBadRequest, "bad request: %v, %v", c.Param("ID"), err)
			return
		}

		conn, err := websocketUpgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			returnInternalServerError(c, err)
			return
		}

		conn.EnableWriteCompression(true)
		eventPricesWS.RegisterNewWriter(eventID, conn)

	})

	router.POST("prices-markets", func(c *gin.Context) {
		requestBody, err := ioutil.ReadAll(c.Request.Body)
		if err != nil {
			returnInternalServerError(c, err)
			return
		}
		err = eventPricesWS.SetIncludeMarket(requestBody)
		jsonResult(c, struct {}{}, err)
	})
}

func setupRouterSports(router *gin.Engine) {
	router.GET("sports", func(c *gin.Context) {
		ch := make(chan eventTypes.Result)
		eventTypes.Get(ch)
		x := <-ch
		close(ch)
		jsonResult(c, x.EventTypes, x.Error)
	})
}

func setupRouterEvents(router *gin.Engine) {
	router.GET("events/*id", func(c *gin.Context) {
		url, urlParsed := utils.QueryUnescape(c.Param("id"))
		eventTypeID, convError := strconv.Atoi(url)
		if !urlParsed || convError != nil {
			c.String(http.StatusBadRequest, "bad request")
			return
		}
		ch := make(chan events.Result)
		events.Get(eventTypeID, ch)
		x := <-ch
		close(ch)
		jsonResult(c, x.Events, x.Error)
	})
}

func setupRouteMarkets(router *gin.Engine) {

	router.GET("event/:ID", func(c *gin.Context) {

		eventID, err := strconv.Atoi(c.Param("ID"))
		if err != nil {
			c.String(http.StatusBadRequest, "bad request: %v, %v", c.Param("ID"), err)
			return
		}

		ch := make(chan event.Result)
		event.Get(eventID, ch)
		x := <-ch
		close(ch)
		jsonResult(c, x.Event, x.Error)
	})
}

func setupRoutePrices(router *gin.Engine) {

	router.GET("prices/:EVENT_ID/*MARKET_IDS", func(c *gin.Context) {

		strEventID := c.Param("EVENT_ID")
		eventID, err := strconv.Atoi(strEventID)
		if err != nil {
			c.String(http.StatusBadRequest, "bad request: EVENT_ID:%v, %v", strEventID, err.Error())
			return
		}

		marketIDs := strings.Split(strings.Trim(c.Param("MARKET_IDS"), "/ "), "/")

		if len(marketIDs) == 0 {
			c.String(http.StatusBadRequest, "bad request: %s, no markets requested", c.Param("MARKET_IDS"))
			return
		}

		ch := make(chan eventPrices.Result)
		eventPrices.Get(eventID, marketIDs, ch)
		x := <-ch
		close(ch)
		jsonResult(c, x.Markets, x.Error)
	})
}

func setupRouteStatic(router *gin.Engine) {
	router.StaticFile("/", "static/index.html")
	router.StaticFile("/index.html", "static/index.html")
	router.Static("/css", "static/css")
	router.Static("/scripts", "static/scripts")

	// тестовые маршруты
	router.StaticFile("test/proxi", "static/proxitest.html")
	router.StaticFile("test/ws", "static/wstest.html")

	router.StaticFile("test1", "static/ts1.html")
}

func jsonResult(c *gin.Context, data interface{}, err error) {

	if err != nil {
		var y struct {
			Error string `json:"error"`
		}
		y.Error = err.Error()
		setCompressedJSON(c, &y)
		return
	}

	var y struct {
		Ok interface{} `json:"ok"`
	}
	y.Ok = data
	setCompressedJSON(c, &y)

}

func setCompressedJSON(c *gin.Context, data interface{}) {
	gz, err := gzip.NewWriterLevel(c.Writer, gzip.DefaultCompression)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	defer gz.Close()

	c.Header("Content-Encoding", "gzip")
	c.Header("Content-Type", "application/json; charset=utf-8")

	encoder := json.NewEncoder(gz)
	encoder.SetIndent("", "    ")

	if err = encoder.Encode(data); err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	c.Writer.WriteHeader(http.StatusOK)
}

func returnInternalServerError(c *gin.Context, err error) {
	c.String(http.StatusInternalServerError, err.Error())
}
