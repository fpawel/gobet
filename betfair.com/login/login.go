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
	Token string
	Error error
}

var muAwaiters sync.RWMutex
var awaiters []chan<- Result
var muSessionToken sync.RWMutex
var sessionToken string
var sessionTime time.Time


type User struct{
	Name string
	Pass string
}

// Login выполняет авторизацию на  betfair.com
func Login(user User) (result Result) {
	const URL = `https://identitysso.betfair.com/api/Login?username=%s&password=%s&Login=true&redirectMethod=POST&product=home.betfair.int&url=https://www.betfair.com/`
	urlStr := fmt.Sprintf(URL, user.Name, user.Pass)
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
	result.Token = m[1]
	return

}

// GetAdminAuth - токен авторизированной сессии betfair.com
func GetAdminAuth(ch chan<- Result) {
	muSessionToken.RLock()
	if sessionToken != "" && time.Since(sessionTime) < 30*time.Minute {
		muSessionToken.RUnlock()
		go func() {
			ch <- Result{Token: sessionToken, Error: nil}
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

		result := Login( User{user, pass})

		s := "successfully"
		if result.Error != nil {
			s = "failed"
		} else {
			muSessionToken.Lock()
			sessionToken = result.Token
			sessionTime = time.Now()
			muSessionToken.Unlock()
		}
		log.Println("Login betfair.com: ", s)

		muAwaiters.Lock()
		defer muAwaiters.Unlock()
		for _, ch := range awaiters {
			ch <- result
		}
		awaiters = nil
	}()
}
