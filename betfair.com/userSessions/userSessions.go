package userSessions

import (
	"gobet/betfair.com/login"
	"sync"
	"time"
)

type session struct {
	token string
	time  time.Time
}

var sessions = make(map[login.User]session)
var mu = sync.RWMutex{}

var awaiters = make(map[login.User][]chan<- login.Result)
var muAwaiters = sync.RWMutex{}

func GetUserSession(user login.User, ch chan<- login.Result) {

	mu.RLock()
	existedSession, hasExistedSession := sessions[user]
	mu.RUnlock()
	if hasExistedSession && time.Since(existedSession.time) < time.Hour {
		ch <- login.Result{existedSession.token, nil}
		return
	}

	muAwaiters.Lock()
	existedAwaiters, hasExistedAwaiters := awaiters[user]
	awaiters[user] = append(existedAwaiters, ch)
	muAwaiters.Unlock()

	if hasExistedAwaiters {
		return
	}

	go loginUser(user)
}

func loginUser(user login.User) {
	r := login.Login(user)
	if r.Error == nil {
		mu.Lock()
		sessions[user] = session{
			token: r.Token,
			time:  time.Now(),
		}
		mu.Unlock()
	}
	muAwaiters.Lock()
	existedAwaiters, _ := awaiters[user]
	for _, ch := range existedAwaiters {
		ch <- r
	}
	delete(awaiters, user)
	muAwaiters.Unlock()
}
