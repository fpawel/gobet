package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"compress/gzip"
	"github.com/gorilla/websocket"

	_ "github.com/user/gobet/betfair.com/aping/client/eventTypes"
	"github.com/user/gobet/proxi"
	"github.com/user/gobet/betfair.com/football/games"
	"github.com/user/gobet/utils"

	"strconv"
	"github.com/user/gobet/betfair.com/aping/client/events"
	"github.com/user/gobet/betfair.com/aping/client/markets"
)

const (
	LOCALHOST_KEY = "LOCALHOST"
)

var footbalGames = games.New()
var websocketUpgrader = websocket.Upgrader{} // use default options

func main() {
	setupRouter( getPort())
}

func setupRouter( port string){
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

	router.GET("football/games", func (c *gin.Context) {

		games, err := footbalGames.Get()
		if err != nil {
			c.String(http.StatusInternalServerError, err.Error())
			return
		}
		setCompressedJSON(c, games)
	})

	setupRouterEvents(router)
	setupRouteMarkets(router )

	setupRouteStatic(router )

	router.Run(":" + port)
}

func setupRouterEvents(router *gin.Engine){
	router.GET("events/*id", func(c *gin.Context){
		url, urlParsed := utils.QueryUnescape(c.Param("id"))
		eventTypeID, convError := strconv.Atoi(url)
		if !urlParsed || convError!=nil {
			c.String(http.StatusBadRequest, "bad request")
			return
		}
		ch := make( chan events.Result )
		events.Get(eventTypeID, ch)
		result := <- ch
		close(ch)
		setCompressedJSON(c, &result)
	})
}
func setupRouteMarkets(router *gin.Engine){

	router.GET("markets/:ID/:needRunners", func(c *gin.Context){

		eventID, convError := strconv.Atoi(c.Param("ID"))
		if convError!=nil {
			c.String(http.StatusBadRequest, "bad request")
			return
		}
		if c.Param("needRunners") != "true" &&
			c.Param("needRunners") != "false" {
			c.String(http.StatusBadRequest, `bad [needRunners] value`)
			return
		}
		needRunners := c.Param("needRunners") == "true"

		ch := make( chan markets.Result )
		markets.Get(eventID, needRunners, ch)
		result := <- ch
		close(ch)

		setCompressedJSON(c, &result)
	})
}

func setupRouteStatic(router *gin.Engine){
	router.StaticFile("/", "static/index.html")
	router.StaticFile("/index.html", "static/index.html")
	router.Static("/css", "static/css")
	router.Static("/scripts", "static/scripts")

	// тестовые маршруты
	router.StaticFile("test/proxi", "static/proxitest.html")
	router.StaticFile("test/ws", "static/wstest.html")
}

func setCompressedJSON(c *gin.Context, data interface{}){
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

func getPort() (port string) {
	port = os.Getenv("PORT")
	if len(os.Args) >= 2 {
		switch strings.ToLower(os.Args[1]) {
		case "localhost":
			port = "8083"
			os.Setenv(LOCALHOST_KEY, "true")
		case "mobileinet":
			port = "8083"
			os.Setenv(LOCALHOST_KEY, "true")
			os.Setenv("MYMOBILEINET", "true")

		case "8083":
			port = "8083"

		default:
			log.Fatalf("wrong argument: %v", os.Args[1])
		}
	}

	log.Printf("port: %s, localhost: %s, mymobileinet: %s",
		port, os.Getenv(LOCALHOST_KEY), os.Getenv("MYMOBILEINET"))
	if port == "" {
		log.Fatal("$PORT must be set")
	}
	return
}



func returnInternalServerError(c *gin.Context, err error) {
	c.String(http.StatusInternalServerError, err.Error())
}





