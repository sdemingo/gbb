package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// Carga el tablón desde la API
func FetchBoard() *Board {
	b := CreateBoard()
	r, err := http.Get(SERVER + "/board")
	if err == nil {
		err = json.NewDecoder(r.Body).Decode(b)
	}
	return b
}

// Carga un thread desde la API
func FetchThread(key string) *Thread {
	th := NewThread("", &Message{})
	r, err := http.Get(SERVER + "/threads/" + key)
	if err == nil {
		err = json.NewDecoder(r.Body).Decode(th)
		return th
	} else {
		return nil
	}
}

// Añade una respuesta a un hilo desde la API
func UpdateThreadWithNewReply(m *Message, key string) error {
	buf := new(bytes.Buffer)
	err := json.NewEncoder(buf).Encode(m)
	_, err = http.NewRequest("UPDATE", SERVER+"/thread/"+key, buf)
	return err
}

// Borra un mensaje desde la Api
func DeleteMessage(m *Message, key string) error {
	client := &http.Client{}
	url := fmt.Sprintf("%s/messages/%d", SERVER, m.Id)
	r, err := http.NewRequest("DELETE", url, nil)
	_, err = client.Do(r)
	return err
}
