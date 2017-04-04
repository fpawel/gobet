package login

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"sync"
	"time"
)

type Result struct {
	SessionToken string
	Error        error
}

var muAwaiters sync.RWMutex
var awaiters []chan<- Result
var muSessionToken sync.RWMutex
var sessionToken string
var sessionTime time.Time

// Login выполняет авторизацию на  betfair.com
func login(user string, pass string) (result Result) {
	const URL = `https://identitysso.betfair.com/api/login?username=%s&password=%s&login=true&redirectMethod=POST&product=home.betfair.int&url=https://www.betfair.com/`
	urlStr := fmt.Sprintf(URL, user, pass)
	var req *http.Request
	if req, result.Error = http.NewRequest("POST", urlStr, nil); result.Error != nil {
		return
	}

	var client http.Client
	var response *http.Response
	if response, result.Error = client.Do(req); result.Error != nil {
		return
	}
	strSetCookie := response.Header.Get("Set-Cookie")

	m := regexp.MustCompile("ssoid=([^;]+);").FindStringSubmatch(strSetCookie)
	if len(m) < 2 {
		result.Error = fmt.Errorf("no headers in response %v", strSetCookie)
		return
	}
	result.SessionToken = m[1]
	return

}

func GetAuth(ch chan<- Result) {
	muSessionToken.RLock()
	if sessionToken != "" && time.Since(sessionTime) < 30*time.Minute {
		muSessionToken.RUnlock()
		go func() {
			ch <- Result{SessionToken: sessionToken, Error: nil}
		}()
		return
	}
	muSessionToken.RUnlock()

	muAwaiters.Lock()
	defer muAwaiters.Unlock()
	awaiters = append(awaiters, ch)
	if len(awaiters) > 1 {
		return
	}
	go func() {
		user := os.Getenv("BETFAIR_LOGIN_USER")
		pass := os.Getenv("BETFAIR_LOGIN_PASS")

		result := login(user, pass)

		s := "successfully"
		if result.Error != nil {
			s = "failed"
		} else {
			muSessionToken.Lock()
			sessionToken = result.SessionToken
			sessionTime = time.Now()
			muSessionToken.Unlock()
		}
		log.Println("login betfair.com: ", s)

		muAwaiters.Lock()
		defer muAwaiters.Unlock()
		for _, ch := range awaiters {
			ch <- result
		}
		awaiters = nil
	}()
}
