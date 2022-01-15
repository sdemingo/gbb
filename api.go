package main

import (
	"encoding/json"
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
		}
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
	}
}

// Recupera todo un hilo
func (a *api) fetchThread(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key := vars["ThreadKey"]
	thread := board.getThread(key)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(thread)
}

func (a *api) fetchBoard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(board)
}

type Server interface {
	Router() http.Handler
}

func NewServer() Server {
	a := &api{}

	r := mux.NewRouter()

	r.HandleFunc("/board", a.fetchBoard).Methods(http.MethodGet)
	r.HandleFunc("/threads/{ThreadKey:[a-zA-Z0-9_]+}", a.fetchThread).Methods(http.MethodGet)
	r.HandleFunc("/threads/{ThreadKey:[a-zA-Z0-9_]+}", a.addMessageToThread).Methods(http.MethodPut)

	// messages:
	r.HandleFunc("/messages/{MsgId:[0-9]+}", a.deleteMessage).Methods(http.MethodDelete)

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
