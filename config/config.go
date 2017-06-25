package config

import (
	"flag"
	"os"
	"fmt"
)

var ReadFromHerokuApp bool
var ConstantlyUpdate bool
var UseBetfairProxi bool
var ControlTraffic bool
var Port string


func init(){
	flag.StringVar(&Port, "port", os.Getenv("PORT"), "HTTP listen spec")
	flag.BoolVar(&ControlTraffic, "ctraf", false, "Control traffic")
	flag.BoolVar(&UseBetfairProxi, "proxi", false, "Use a proxy server to access betfair.com")
	flag.BoolVar(&ReadFromHerokuApp, "hrkf", false, "Read the list of football matches through my heroku application")
	flag.BoolVar(&ConstantlyUpdate , "updf", true, "Constantly update the list of football matches")
	flag.Parse()
	fmt.Println("ReadFromHerokuApp:",ReadFromHerokuApp)
	fmt.Println(" ConstantlyUpdate:",ConstantlyUpdate)
	fmt.Println("  UseBetfairProxi:",UseBetfairProxi)
	fmt.Println("   ControlTraffic:",ControlTraffic)
	fmt.Println("             Port:",Port)

}
