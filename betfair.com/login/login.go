package login

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
)



// Login выполняет вход на betfair.com, возвращает строку сессии
func Login(user string, pass string) (sessionToken string, err error) {
	const URL = `https://identitysso.betfair.com/api/login?username=%s&password=%s&login=true&redirectMethod=POST&product=home.betfair.int&url=https://www.betfair.com/`
	urlStr := fmt.Sprintf(URL, user, pass)
	var req *http.Request
	if req, err = http.NewRequest("POST", urlStr, nil); err != nil {
		return
	}

	var client http.Client
	var response *http.Response
	if response, err = client.Do(req); err != nil {
		return
	}
	strSetCookie := response.Header.Get("Set-Cookie")

	m := regexp.MustCompile("ssoid=([^;]+);").FindStringSubmatch(strSetCookie)
	if len(m) < 2 {
		err = fmt.Errorf("no headers in response %v", strSetCookie)
		return
	}
	sessionToken = m[1]
	return

}

// SessionToken строка сессии betfair.com
func SessionToken() string {
	return sessionToken
}

var sessionToken string

func init() {
	user := os.Getenv("BETFAIR_LOGIN_USER")
	pass := os.Getenv("BETFAIR_LOGIN_PASS")
	var err error
	sessionToken, err = Login(user, pass)
	if err != nil {
		log.Fatalf("can`t login betfair.com: %v", err)
	}
	log.Printf("login betfair.com: ok")

}
