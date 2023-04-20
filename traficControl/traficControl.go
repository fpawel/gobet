package traficControl

import (
	"gobet/config"
	"gobet/utils"
	"log"
	"sync"
)

// AddTotalBytesRead логгирует текущее значение количества считанных мегабайт
func AddTotalBytesRead(value int, who string) {
	if !config.Get().ControlTraffic {
		return
	}
	mu.Lock()
	nextTotalBytesRead := totalBytesRead + uint64(value)
	totalBytesRead = nextTotalBytesRead
	mu.Unlock()
	log.Println("Control traffic:", who, "+", utils.HumanizeBytes(uint64(value)), ":",
		utils.HumanizeBytes(nextTotalBytesRead))
}

var (
	totalBytesRead uint64
	mu             sync.Mutex
)
