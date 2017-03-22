package envvars

import (
	"log"
	"os"
	"strings"
)

const (
	VarLocalHost    = "LOCALHOST"
	VarMyMobileInet = "MYMOBILEINET"
	VarPort         = "PORT"
	VarRunFootbal   = "RUN_FOOTBALL"
)

// MobileInet true если переменная окружение _MYMOBILEINET утановлена в true
func MobileInet() bool {
	return os.Getenv(VarMyMobileInet) == "true"
}

func Port() string {
	return os.Getenv(VarPort)
}

func Localhost() bool {
	return os.Getenv(VarLocalHost) == "true"
}

func RunFootbal() bool {
	return os.Getenv(VarRunFootbal) == "true"
}

func init() {
	os.Setenv(VarRunFootbal, "true")

	if len(os.Args) == 2 {
		switch strings.ToLower(os.Args[1]) {
		case "localhost":
			os.Setenv(VarPort, "8083")
			os.Setenv(VarLocalHost, "true")
		case "mobileinet":
			os.Setenv(VarPort, "8083")
			os.Setenv(VarLocalHost, "true")
			os.Setenv(VarMyMobileInet, "true")
		case "8083":
			os.Setenv(VarPort, "8083")
		case "nofootball":
			os.Setenv(VarRunFootbal, "false")
		default:
			log.Fatalf("wrong argument: %v", os.Args[1])
		}
	}
	log.Printf("port: %s, localhost: %s, mymobileinet: %s",
		os.Getenv(VarPort),
		os.Getenv(VarLocalHost), os.Getenv("_MYMOBILEINET"))
	if os.Getenv(VarPort) == "" {
		log.Fatal("$_PORT must be set")
	}
}
