package config

import (
	"flag"
	"os"
	"fmt"
)

var c C

type C struct {
	ReadFromHerokuApp bool
	ConstantlyUpdate bool
	UseBetfairProxi bool
	ControlTraffic bool
	Port string
}

func Get() C {
	return  c
}


func init(){
	flag.StringVar(&c.Port, "port", os.Getenv("PORT"), "HTTP listen spec")
	flag.BoolVar(&c.ControlTraffic, "ctraf", false, "Control traffic")
	flag.BoolVar(&c.UseBetfairProxi, "proxi", false, "Use a proxy server to access betfair.com")
	flag.BoolVar(&c.ReadFromHerokuApp, "hrkf", false, "Read the list of football matches through my heroku application")
	flag.BoolVar(&c.ConstantlyUpdate, "updf", true, "Constantly update the list of football matches")
	flag.Parse()
	fmt.Println("ReadFromHerokuApp:",c.ReadFromHerokuApp)
	fmt.Println(" ConstantlyUpdate:",c.ConstantlyUpdate)
	fmt.Println("  UseBetfairProxi:",c.UseBetfairProxi)
	fmt.Println("   ControlTraffic:",c.ControlTraffic)
	fmt.Println("             Port:",c.Port)

}
