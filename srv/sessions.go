package srv

import (
	"net/http"
	"time"
)

type Session struct {
	Id    string
	User  string
	Stamp time.Time
}

var sessionCache map[string]*Session

func CreateSession(login string) *Session {
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
		return board.GetUser(session.User)
	}
	return nil
}
