package traficControl

import (
	"log"
	"sync"
	"github.com/user/gobet/utils"
	"github.com/user/gobet/config"
)


// AddTotalBytesReaded логгирует текущее значение количества считанных мегабайт
func AddTotalBytesReaded(value int, who string) {
	if !config.Get().ControlTraffic {
		return
	}
	mu.Lock()
	nextTotalBytesReaded := totalBytesReaded + uint64(value)
	totalBytesReaded = nextTotalBytesReaded
	mu.Unlock()
	log.Println( "Control traffic:", who, "+", utils.HumanateBytes(uint64(value)), ":",
		utils.HumanateBytes(nextTotalBytesReaded) )
}

var totalBytesReaded uint64
var mu sync.Mutex


