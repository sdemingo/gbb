package srv

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

type Session struct {
	Id    string
	User  string
	Stamp time.Time
}

const SESSION_TIMEALIVE = 30.0 //minutes

var sessionCache map[string]*Session
var sessionMutex sync.Mutex

func InitSessionCache() {
	sessionCache = make(map[string]*Session)
	go sessionRoutine()
}

func CreateSession(login string) *Session {
	sessionMutex.Lock()
	defer sessionMutex.Unlock()

	s := new(Session)
	s.Id = RandomString(64)
	s.User = login
	s.Stamp = time.Now()
	sessionCache[s.Id] = s
	return s
}

func GetUserFromSession(r *http.Request) *User {
	token, err := r.Cookie("token")
	if err != nil {
		return nil
	}
	session := sessionCache[token.Value]
	if session != nil {
		session.Stamp = time.Now()
		return board.GetUser(session.User)
	}
	return nil
}

func sessionRoutine() {
	for {
		sessionMutex.Lock()
		defer sessionMutex.Unlock()

		for _, session := range sessionCache {
			now := time.Now()
			diff := now.Sub(session.Stamp)

			if diff.Minutes() >= SESSION_TIMEALIVE {
				delete(sessionCache, session.Id)
				logEvent(fmt.Sprintf("Sesión de %s eliminada por inactividad\n", session.User))
			}
		}
		sessionMutex.Unlock()
		time.Sleep(10 * time.Second)
	}
}
