package srv

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/gorilla/mux"
)

type api struct {
	router http.Handler
}

// Borra un mensaje del servidor
func (a *api) deleteMessage(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromSession(r)
	if user != nil {
		vars := mux.Vars(r)
		id, err := strconv.Atoi(vars["MsgId"])
		if err == nil {
			m := board.getMessage(id)
			if m == nil {
				a.jsonerror(w, "Bad msg id", 404)
				return
			}

			if m.Author != user.Login && !user.IsAdmin {
				a.jsonerror(w, "Bad msg id or bad msg author", 404)
				return
			}

			if th := m.Parent; th != nil {
				err = m.DeleteFromBD()
				if err != nil {
					logEvent(fmt.Sprintf("BD ERROR: Falló el borrado del mensaje [%d] del hilo %s por %s: %s", m.Id, th.Id, user.Login, err))
				}
				err = th.delMessage(m)
				if err != nil {
					logEvent(fmt.Sprintf("Falló el borrado del mensaje [%d] del hilo %s por %s: %s", m.Id, th.Id, user.Login, err))
				} else {
					logEvent(fmt.Sprintf("Se ha borrado el mensaje [%d]  del hilo %s por %s", m.Id, th.Id, user.Login))
				}

			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			json.NewEncoder(w).Encode(m)

		} else {
			a.jsonerror(w, "Bad msg id", 404)
		}
	} else {
		a.jsonerror(w, "Usuario no autenticado. Token desconocido", 404)
	}
}

// Actualiza el contenido de un mensaje en el servidor
func (a *api) updateMessageInThread(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromSession(r)
	if user != nil {
		vars := mux.Vars(r)
		id, err := strconv.Atoi(vars["MsgId"])
		if err == nil {
			if storedMsg := board.getMessage(id); storedMsg != nil && storedMsg.Author == user.Login {
				auxMsg := NewMessage("", "")
				err := json.NewDecoder(r.Body).Decode(auxMsg)
				if err == nil {
					storedMsg.Text = auxMsg.Text
					err=storedMsg.Save(true)
                    if err!=nil{
						logEvent(fmt.Sprintf("BD ERROR: Falló el actualizado del mensaje [%d]: %s", storedMsg.Id, err))
					}
					storedMsg.Text = auxMsg.Text
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(storedMsg)
				} else {
					a.jsonerror(w, fmt.Sprintf("%s", err), 404)
				}
			} else {
				a.jsonerror(w, "Unknow msg id or bad author", 404)
			}
		} else {
			a.jsonerror(w, "Bad msg id", 404)
		}
	} else {
		a.jsonerror(w, "Usuario no autenticado. Token desconocido", 404)
	}
}

// Añade un mensaje al servidor
func (a *api) addMessageToThread(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromSession(r)
	if user != nil {
		vars := mux.Vars(r)
		key := vars["ThreadKey"]
		thread := board.getThread(key)
		if thread.IsClosed {
			a.jsonerror(w, "Error: Thread is closed", 404)
			return
		}
		m := NewMessage("", "")
		err := json.NewDecoder(r.Body).Decode(m)
		if err == nil && thread != nil && m != nil {
			m.Parent = thread
			m.Author = user.Login
			err = m.Save(false)
			if err != nil {
				logEvent(fmt.Sprintf("BD ERROR: Falló añadir el mensaje [%d] al hilo %s por %s: %s", m.Id, thread.Id, user.Login, err))
				a.jsonerror(w, "Operation failed", 404)
				return
			}
			m.Parent.addMessage(m)
			logEvent(fmt.Sprintf("%s ha añadido el mensaje [%d] al hilo %s", user.Login, m.Id, thread.Id))
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(m)
		} else {
			a.jsonerror(w, "Bad request payload", 404)
		}
	} else {
		a.jsonerror(w, "Usuario no autenticado. Token desconocido", 404)
	}
}

// Recupera todo un hilo
func (a *api) fetchThread(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromSession(r)
	if user != nil {
		vars := mux.Vars(r)
		key := vars["ThreadKey"]
		thread := board.getThread(key)
		if thread != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(thread)
		} else {
			a.jsonerror(w, "Id de hilo desconocida", 404)
		}
	} else {
		a.jsonerror(w, "Usuario no autenticado. Token desconocido", 404)
	}
}

// Borra todo el hilo completo
func (a *api) deleteThread(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromSession(r)
	if user != nil {
		vars := mux.Vars(r)
		key := vars["ThreadKey"]
		thread := board.getThread(key)
		if (thread.Author != user.Login) && (!user.IsAdmin) {
			a.jsonerror(w, "Operación no autorizada", 404)
			return
		}
		if thread != nil {
			err:=thread.Delete()
			board.delThread(thread)
			if err!=nil{
				logEvent(fmt.Sprintf("BD ERROR: Fallo el borrado del hilo %s por parte de %s", thread.Id, user.Login))
			}else{
				logEvent(fmt.Sprintf("%s ha borrado el hilo %s", user.Login, thread.Id))
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(thread)
		} else {
			a.jsonerror(w, "Unknow thread key", 404)
		}
	} else {
		a.jsonerror(w, "Usuario no autenticado. Token desconocido", 404)
	}
}

// Cierra un hilo para evitar que tenga más respuestas. Si el parámetro close
// es igual a 0 el hilo se abre. Si distinto de 0 quedará cerrado
func (a *api) operateWithThread(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromSession(r)
	if user != nil {
		vars := mux.Vars(r)
		key := vars["ThreadKey"]
		command := vars["Cmd"]
		thread := board.getThread(key)
		if thread != nil && user.IsAdmin {
			if command == "close" || command == "open" {
				thread.IsClosed = (command == "close")
			}

			if command == "fixed" || command == "free" {
				thread.IsFixed = (command == "fixed")
			}
			err := thread.Update()
			if err!=nil{
				logEvent(fmt.Sprintf("BD ERROR: Falló actualizar el modo del hilo hilo %s: %s", thread.Id,err))
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(thread)
		} else {
			a.jsonerror(w, "Bad thread key or bad user", 404)
		}
	} else {
		a.jsonerror(w, "Usuario no autenticado. Token desconocido", 404)
	}
}

// Recupera el tablón
func (a *api) fetchBoard(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromSession(r)
	if user != nil {
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(board)
	} else {
		a.jsonerror(w, "Usuario no autenticado. Token desconocido", 404)
	}
}

// Recupera los hilos que contengan el patrón que viaja en el payload
func (a *api) filterBoard(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromSession(r)
	if user != nil {
		vars := mux.Vars(r)
		pattern := vars["Pattern"]
		filteredThreads := board.filterThreads(pattern)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(filteredThreads)
	} else {
		a.jsonerror(w, "Usuario no autenticado. Token desconocido", 404)
	}
}

// Crea un nuevo thread (sin mensaje principal). El primer mensaje debe
// legar a través de otra llamada que debería recibirse tra esta
func (a *api) addThreadToBoard(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromSession(r)
	if user != nil {
		var title string
		json.NewDecoder(r.Body).Decode(&title)
		th := NewThread(title, nil)
		th.Author = user.Login
		board.addThread(th)
		err:=th.Save()
		if err!=nil{
			logEvent(fmt.Sprintf("BD ERROR: Fallo añdir hilo %s por parte del usuario %s", th.Id, user.Login))
		}else{
			logEvent(fmt.Sprintf("%s ha añadido el hilo %s", user.Login, th.Id))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(th)
	} else {
		a.jsonerror(w, "Usuario no autenticado. Token desconocido", 404)
	}
}

// Retorna la info de un usuario
func (a *api) getUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	login := vars["Login"]
	var u *User
	if u = board.GetUser(login); u == nil {
		a.jsonerror(w, "User not exists in the database", 404)
		return
	}
	w.WriteHeader(200)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(u)
}

// Verifica las credenciales de un usuario
func (a *api) verifyUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	login := vars["Login"]
	var u *User
	if u = board.GetUser(login); u == nil {
		logEvent(fmt.Sprintf("Se intenta acceder con usuario desconocido: %s", login))
		a.jsonerror(w, "User not exists in the database", 404)
		return
	}
	pass_s := ""
	err := json.NewDecoder(r.Body).Decode(&pass_s)
	if err != nil {
		logEvent(fmt.Sprintf("Se recibe mensaje con credenciales corrupto"))
		a.jsonerror(w, "Bad createUser payload", 404)
		return
	}

	if strings.Compare(fmt.Sprintf("%s", pass_s), fmt.Sprintf("%s", u.Password)) == 0 {
		// Auth OK. Create session and send response with token
		s := CreateSession(login)
		w.Header().Set("Content-Type", "application/json")
		logEvent(fmt.Sprintf("%s ha iniciado sesión", login))
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(s.Id)
	} else {
		a.jsonerror(w, "Bad password", 404)
	}
}

func (a *api) changePassword(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromSession(r)
	if user != nil {
		newpass_s := ""
		err := json.NewDecoder(r.Body).Decode(&newpass_s)
		if err != nil {
			a.jsonerror(w, "Bad createUser payload", 404)
			return
		}
		user.Password = []byte(newpass_s)
		user.Save(true)
		logEvent(fmt.Sprintf("%s ha actualizado su contraseña", user.Login))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(user)
	} else {
		a.jsonerror(w, "Usuario no autenticado. Token desconocido", 404)
	}
}

// Recarga toda la tabla de usuarios
func (a *api) reloadUsers(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromSession(r)
	if user != nil && user.IsAdmin {
		board.LoadUsers()
		logEvent("Se carga tabla de usuarios en el servidor")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		json.NewEncoder(w).Encode("")
	} else {
		a.jsonerror(w, "Usuario no autenticado. Token desconocido", 404)
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

	// board:
	r.HandleFunc("/board", a.fetchBoard).Methods(http.MethodGet)
	r.HandleFunc("/board", a.addThreadToBoard).Methods(http.MethodPost)
	r.HandleFunc("/board/{Pattern:[a-zA-Z0-9_]+}", a.filterBoard).Methods(http.MethodGet)
	r.HandleFunc("/board/users/reload", a.reloadUsers).Methods(http.MethodGet)

	// threads:
	r.HandleFunc("/threads/{ThreadKey:[a-zA-Z0-9_]+}", a.fetchThread).Methods(http.MethodGet)
	r.HandleFunc("/threads/{ThreadKey:[a-zA-Z0-9_]+}", a.addMessageToThread).Methods(http.MethodPut)
	r.HandleFunc("/threads/{ThreadKey:[a-zA-Z0-9_]+}/{Cmd:[a-z]+}", a.operateWithThread).Methods(http.MethodPut)
	r.HandleFunc("/threads/{ThreadKey:[a-zA-Z0-9_]+}", a.deleteThread).Methods(http.MethodDelete)

	// messages:
	r.HandleFunc("/messages/{MsgId:[0-9]+}", a.deleteMessage).Methods(http.MethodDelete)
	r.HandleFunc("/messages/{MsgId:[0-9]+}", a.updateMessageInThread).Methods(http.MethodPut)

	// users:
	r.HandleFunc("/users/{Login:[a-zA-Z0-9_]+}", a.verifyUser).Methods(http.MethodPost)
	r.HandleFunc("/users/{Login:[a-zA-Z0-9_]+}", a.getUser).Methods(http.MethodGet)
	r.HandleFunc("/users/{Login:[a-zA-Z0-9_]+}/changePassword", a.changePassword).Methods(http.MethodPut)

	a.router = r
	return a
}

func (a *api) Router() http.Handler {
	return a.router
}

var board *Board
var mutex sync.Mutex


func GetConnection() (*sql.DB,error) {
	
	mutex.Lock()
	db, err := sql.Open("sqlite3", dbPathFile)
	if err != nil {
		mutex.Unlock()
		return nil, err
	}
	return db,nil
}

func CloseConnection(db *sql.DB){

	mutex.Unlock();
	db.Close()
}

const PORT = 8080

var dbPathFile = "../data/gbb.db"
var logFile *os.File
var logFileName = "gbb.log"
var logDirectory = "/var/log/gbb"
var SERVER = "http://localhost:" + fmt.Sprintf("%d", PORT)

func InitLog(enable bool) {
	if enable {
		err := os.MkdirAll(logDirectory, os.ModePerm)
		if err != nil {
			return
		}
		logFile, err := os.OpenFile(filepath.Join(logDirectory, logFileName), os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0660)
		if err != nil {
			panic(err)
		}
		log.SetOutput(logFile)
	} else {
		log.SetOutput(ioutil.Discard)
	}
}

func logEvent(text string) {
	log.Printf("%s\n", text)
}

func ServerInit(dir string) {

	dbPathFile = filepath.Join(dir, dbPathFile)
	board = CreateBoard()

	InitLog(true)

	logEvent("GBB Loading database from " + dbPathFile + " ...")
	err := board.Load()
	if err != nil {
		logEvent("Error: Database not found. You must execute initdb to create the database file")
		os.Exit(-1)
	}
	logEvent("GBB Server running ...")

	InitSessionCache()

	s := NewServer()
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", PORT), s.Router()))
}
