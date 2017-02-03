package eventTypes

import (
	"encoding/json"
	"fmt"
	"github.com/user/gobet/betfair.com/aping/client"
	"github.com/user/gobet/betfair.com/aping/client/appkey"
	"github.com/user/gobet/betfair.com/aping/client/endpoint"
	"log"
	"os"
	"strconv"
)

var data []client.EventType

// Get возвращает список типов событий betfair
func Get() []client.EventType {
	return data
}

func init() {

	var reqParams struct {
		Locale string   `json:"locale"`
		Filter struct{} `json:"filter"`
	}
	reqParams.Locale = "ru"
	ep := endpoint.BettingAPI("listEventTypes")
	responseBody, err := appkey.GetResponse(ep, &reqParams)
	if err != nil {
		return
	}
	var xs struct {
		R []struct {
			MarketCount int `json:"marketCount"`
			X           struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"eventType"`
		} `json:"result"`
	}
	err = json.Unmarshal(responseBody, &xs)
	if err != nil {
		log.Fatalf("can`t get event types from %v: %v", string(responseBody), err)
	}
	for _, x := range xs.R {
		id, err := strconv.Atoi(x.X.ID)
		if err != nil {
			log.Fatalf("wrong event type id %v: %v", x, err)
		}
		y := client.EventType{ID: id, Name: x.X.Name, MarketCount: x.MarketCount}
		data = append(data, y)
		log.Printf("sport: %v\n", y)
	}

	filePath := "./static/scripts/sports.js"
	fs, err := os.Create(filePath)
	if err != nil {
		log.Fatalf("%s: %v", filePath, err)
	}
	defer fs.Close()

	bytes, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		log.Fatalf("%v", err)
	}
	fmt.Fprintf(fs, "sports = %s", string(bytes))
}
