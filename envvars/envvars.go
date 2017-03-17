package envvars

import (
	"os"
	"strings"
	"log"
)

const (
	_LOCALHOST    = "LOCALHOST"
	_MYMOBILEINET = "MYMOBILEINET"
	_PORT         = "PORT"
)


// MobileInet true если переменная окружение _MYMOBILEINET утановлена в true
func MobileInet() bool {
	return os.Getenv(_MYMOBILEINET) == "true"
}

func Port() string {
	return os.Getenv(_PORT)
}

func Localhost() bool {
	return os.Getenv(_LOCALHOST) == "true"
}

func init(){

	if len(os.Args) >= 2 {
		switch strings.ToLower(os.Args[1]) {
		case "localhost":
			os.Setenv(_PORT, "8083")
			os.Setenv(_LOCALHOST, "true")
		case "mobileinet":
			os.Setenv(_PORT, "8083")
			os.Setenv(_LOCALHOST, "true")
			os.Setenv(_MYMOBILEINET, "true")
		case "8083":
			os.Setenv(_PORT, "8083")
		default:
			log.Fatalf("wrong argument: %v", os.Args[1])
		}
	}
	log.Printf("port: %s, localhost: %s, mymobileinet: %s",
		os.Getenv(_PORT),
		os.Getenv(_LOCALHOST), os.Getenv("_MYMOBILEINET"))
	if os.Getenv(_PORT) == "" {
		log.Fatal("$_PORT must be set")
	}
}
