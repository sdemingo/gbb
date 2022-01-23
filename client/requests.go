package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"gbb/srv"
	"net/http"
	"sort"
)

/*

	Client Requests

*/

var client = &http.Client{}

// Carga el tablón desde la API
func FetchBoard() *srv.Board {
	b := srv.CreateBoard()
	r, err := http.Get(srv.SERVER + "/board")
	if err == nil {
		err = json.NewDecoder(r.Body).Decode(b)
	}
	sort.Sort(b)
	return b
}

// Carga un thread desde la API
func FetchThread(key string) *srv.Thread {
	th := srv.NewThread("", &srv.Message{})
	r, err := http.Get(srv.SERVER + "/threads/" + key)
	if err == nil {
		err = json.NewDecoder(r.Body).Decode(th)
		for i := range th.Messages {
			th.Messages[i].Parent = th
		}
		return th
	} else {
		return nil
	}
}

// Manda un patrón a la API para recuperar los hilos que
// lo contengan
func FindThreads(pattern string) []*srv.Thread {
	matches := make([]*srv.Thread, 0)
	r, err := http.Get(srv.SERVER + "/board/" + pattern)
	if err == nil {
		err = json.NewDecoder(r.Body).Decode(&matches)
	}
	return matches
}

// Crea un nuevo hilo a través de la API. Envía el título del thread y retorna la
// clave del thread creado
func CreateThread(title string) (th *srv.Thread, err error) {
	buf := new(bytes.Buffer)
	err = json.NewEncoder(buf).Encode(title)
	url := fmt.Sprintf("%s/board", srv.SERVER)
	r, err := http.NewRequest("POST", url, buf)
	resp, err := client.Do(r)

	th = new(srv.Thread)
	err = json.NewDecoder(resp.Body).Decode(th)
	return th, err
}

// Borra un hilo completo a través de la API
func DeleteThread(th *srv.Thread) error {
	url := fmt.Sprintf("%s/threads/%s", srv.SERVER, th.Id)
	r, err := http.NewRequest("DELETE", url, nil)
	_, err = client.Do(r)
	return err
}

// Añade una respuesta a un hilo desde la API
func UpdateThreadWithNewReply(m *srv.Message, key string) error {
	buf := new(bytes.Buffer)
	err := json.NewEncoder(buf).Encode(m)
	url := fmt.Sprintf("%s/threads/%s", srv.SERVER, key)
	r, err := http.NewRequest("PUT", url, buf)
	_, err = client.Do(r)
	return err
}

// Actualiza el estado de un thread en base al cmd enviado. El valor de
// cmd puede ser: open|close|fixed|free
func UpdateThreadStatus(th *srv.Thread, cmd string) error {
	url := fmt.Sprintf("%s/threads/%s/%s", srv.SERVER, th.Id, cmd)
	r, err := http.NewRequest("PUT", url, nil)
	_, err = client.Do(r)
	return err
}

// Actualiza el contenido de un mensaje
func UpdateContentMessage(m *srv.Message) error {
	buf := new(bytes.Buffer)
	err := json.NewEncoder(buf).Encode(m)
	url := fmt.Sprintf("%s/messages/%d", srv.SERVER, m.Id)
	r, err := http.NewRequest("PUT", url, buf)
	_, err = client.Do(r)
	return err
}

// Borra un mensaje desde la Api
func DeleteMessage(m *srv.Message, key string) error {
	url := fmt.Sprintf("%s/messages/%d", srv.SERVER, m.Id)
	r, err := http.NewRequest("DELETE", url, nil)
	_, err = client.Do(r)
	return err
}

// Retorna la info del usuario o nil si el usuario no existe
func FetchUser(login string) *srv.User {
	user := new(srv.User)
	r, err := http.Get(srv.SERVER + "/users/" + login)
	if err == nil {
		err = json.NewDecoder(r.Body).Decode(&user)
		return user
	}
	return nil
}

// Envía la password y el login y recibe el token de sesión del usuario
func AuthUser(login string, password string) string {
	buf := new(bytes.Buffer)
	err := json.NewEncoder(buf).Encode(password) // pasar el hash
	url := fmt.Sprintf("%s/users/%s", srv.SERVER, login)
	r, err := http.NewRequest("POST", url, buf)
	resp, err := client.Do(r)
	token := ""
	if err == nil && resp.Status == "200 OK" {
		err = json.NewDecoder(resp.Body).Decode(&token)
		return token
	}
	return token
}
