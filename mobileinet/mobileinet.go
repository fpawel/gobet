package mobileinet

import (
	"log"
	"sync"
	"github.com/user/gobet/utils"
	"github.com/user/gobet/envvars"
)


// LogAddTotalBytesReaded логгирует текущее значение количества считанных мегабайт
func LogAddTotalBytesReaded(value int, who string) {
	if !envvars.MobileInet() {
		return
	}
	mu.Lock()
	nextTotalBytesReaded := totalBytesReaded + uint64(value)
	totalBytesReaded = nextTotalBytesReaded
	mu.Unlock()
	log.Println("MOBILE INET:", who, "+", utils.HumanateBytes(uint64(value)), ":",
		utils.HumanateBytes(nextTotalBytesReaded) )
}

var totalBytesReaded uint64
var mu sync.Mutex


