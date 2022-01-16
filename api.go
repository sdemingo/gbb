package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

const SERVER = "http://localhost:8080"

type api struct {
	router http.Handler
}

// Borra un mensaje del servidor
func (a *api) deleteMessage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["MsgId"])
	if err == nil {
		if m := board.getMessage(id); m != nil {
			if th := m.Parent; th != nil {
				m.DeleteFromBD()
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusAccepted)
			json.NewEncoder(w).Encode(m)
		} else {
			a.jsonerror(w, "Unknow msg id", 404)
			return
		}
	} else {
		a.jsonerror(w, "Bad msg id", 404)
	}
}

// Actualiza el contenido de un mensaje en el servidor
func (a *api) updateMessageInThread(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["MsgId"])
	if err == nil {
		if storedMsg := board.getMessage(id); storedMsg != nil {
			auxMsg := NewMessage("", "")
			err := json.NewDecoder(r.Body).Decode(auxMsg)
			if err == nil {
				storedMsg.Text = auxMsg.Text
				storedMsg.Save(true)
				storedMsg.Text = auxMsg.Text
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(storedMsg)
			} else {
				a.jsonerror(w, fmt.Sprintf("%s", err), 404)
				return
			}
		} else {
			a.jsonerror(w, "Unknow msg id", 404)
			return
		}
	} else {
		a.jsonerror(w, "Bad msg id", 404)
	}
}

// Añade un mensaje al servidor
func (a *api) addMessageToThread(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key := vars["ThreadKey"]
	thread := board.getThread(key)
	m := NewMessage("", "")
	err := json.NewDecoder(r.Body).Decode(m)
	if err == nil {
		m.Parent = thread
		m.Save(false)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(m)
	} else {
		a.jsonerror(w, "Bad request payload", 404)
	}
}

// Recupera todo un hilo
func (a *api) fetchThread(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key := vars["ThreadKey"]
	thread := board.getThread(key)
	if thread != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(thread)
	} else {
		a.jsonerror(w, "Unknow thread key", 404)
	}
}

// Borra todo el hilo completo
func (a *api) deleteThread(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key := vars["ThreadKey"]
	thread := board.getThread(key)
	if thread != nil {
		thread.Delete()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(thread)
	} else {
		a.jsonerror(w, "Unknow thread key", 404)
	}
}

// Cierra un hilo para evitar que tenga más respuestas. Si el parámetro close
// es igual a 0 el hilo se abre. Si distinto de 0 quedará cerrado
func (a *api) operateWithThread(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key := vars["ThreadKey"]
	command := vars["Close"]
	thread := board.getThread(key)
	if thread != nil {
		if command == "close" || command == "open" {
			thread.IsClosed = (command == "close")
		}

		if command == "fixed" || command == "free" {
			thread.IsFixed = (command == "fixed")
		}
		thread.Update()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(thread)
	} else {
		a.jsonerror(w, "Unknow thread key", 404)
	}
}

// Recupera el tablón
func (a *api) fetchBoard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(board)
}

// Crea un usuario
func (a *api) createUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	login := vars["Login"]
	if u := board.GetUser(login); u != nil {
		a.jsonerror(w, "User exists in the database", 404)
		return
	}
	pass_s, err := io.ReadAll(r.Body)
	if err != nil {
		a.jsonerror(w, "Bad createUser payload", 404)
		return
	}
	u := NewUser(login, pass_s)
	u.Save(false)
	board.AddUser(u)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode("OK")
}

// Verifica las credenciales de un usuario
func (a *api) verifyUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	login := vars["Login"]
	var u *User
	if u = board.GetUser(login); u == nil {
		a.jsonerror(w, "User not exists in the database", 404)
		return
	}
	pass_s, err := io.ReadAll(r.Body)
	if err != nil {
		a.jsonerror(w, "Bad createUser payload", 404)
		return
	}

	if bytes.Compare(pass_s, u.Password) == 0 {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode("OK")
	} else {
		a.jsonerror(w, "Bad password", 404)
	}

}

// Genera respuesta de error
func (a *api) jsonerror(w http.ResponseWriter, err interface{}, code int) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(err)
}

type Server interface {
	Router() http.Handler
}

func NewServer() Server {
	a := &api{}

	r := mux.NewRouter()

	// threads:
	r.HandleFunc("/board", a.fetchBoard).Methods(http.MethodGet)
	r.HandleFunc("/threads/{ThreadKey:[a-zA-Z0-9_]+}", a.fetchThread).Methods(http.MethodGet)
	r.HandleFunc("/threads/{ThreadKey:[a-zA-Z0-9_]+}", a.addMessageToThread).Methods(http.MethodPut)
	r.HandleFunc("/threads/{ThreadKey:[a-zA-Z0-9_]+}/{Close:[a-z]+}", a.operateWithThread).Methods(http.MethodPut)
	r.HandleFunc("/threads/{ThreadKey:[a-zA-Z0-9_]+}", a.deleteThread).Methods(http.MethodDelete)

	// messages:
	r.HandleFunc("/messages/{MsgId:[0-9]+}", a.deleteMessage).Methods(http.MethodDelete)
	r.HandleFunc("/messages/{MsgId:[0-9]+}", a.updateMessageInThread).Methods(http.MethodPut)

	// users:
	r.HandleFunc("/users/{Login:[a-zA-Z0-9_]+}", a.verifyUser).Methods(http.MethodPost)
	r.HandleFunc("/users/{Login:[a-zA-Z0-9_]+}/new", a.createUser).Methods(http.MethodPost)

	a.router = r
	return a
}

func (a *api) Router() http.Handler {
	return a.router
}

func ServerInit() {
	s := NewServer()
	log.Fatal(http.ListenAndServe(":8080", s.Router()))
}
