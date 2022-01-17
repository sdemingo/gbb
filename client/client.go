package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"gbb/srv"
	"log"
	"net/http"
	"os"
	"os/user"
	"sort"
	"strings"

	"github.com/gdamore/tcell"
)

// Carga el tablón desde la API
func FetchBoard() *srv.Board {
	b := srv.CreateBoard()
	r, err := http.Get(srv.SERVER + "/board")
	if err == nil {
		err = json.NewDecoder(r.Body).Decode(b)
	}
	return b
}

// Carga un thread desde la API
func FetchThread(key string) *srv.Thread {
	th := srv.NewThread("", &srv.Message{})
	r, err := http.Get(srv.SERVER + "/threads/" + key)
	if err == nil {
		err = json.NewDecoder(r.Body).Decode(th)
		return th
	} else {
		return nil
	}
}

// Añade una respuesta a un hilo desde la API
func UpdateThreadWithNewReply(m *srv.Message, key string) error {
	buf := new(bytes.Buffer)
	err := json.NewEncoder(buf).Encode(m)
	_, err = http.NewRequest("UPDATE", srv.SERVER+"/thread/"+key, buf)
	return err
}

// Borra un mensaje desde la Api
func DeleteMessage(m *srv.Message, key string) error {
	client := &http.Client{}
	url := fmt.Sprintf("%s/messages/%d", srv.SERVER, m.Id)
	r, err := http.NewRequest("DELETE", url, nil)
	_, err = client.Do(r)
	return err
}

const HELP_TEXT = `

	Índice de teclas y comandos
	===========================

	a      -    Añade un hilo o un mensaje
	d      -    Borrar un hilo o un mensaje
	e      -    Editar un mensaje
	b      -    Buscar hilos por palabras clave
	↑↓     -    Navegar entre hilos o mensajes
	AvPg   - 	Avanzar página de un mensaje
	RePg   -    Retroceder página de un mensaje
	ESC    -    Ir a la ventana anterior
	?      -    Mostrar este mensaje de ayuda
	f      -    Fijar un hilo en la cabecera. Solo para administradores
	c      -    Cerrar un hilo para nuevas respuestas. Solo para administradores



	Acerca de GBB
	=============

	GBB ha sido licenciado con GNU GENERAL PUBLIC LICENSE Version 3
	Su código está disponible en: https://github.com/sdemingo/gbb
`

var APP_TITLE = "GBB v0.1"
var DefaultStyle tcell.Style
var Username string
var isAdmin bool
var activeMode = 0
var lastActiveMode = 0
var uiChannel chan int
var confirmDelete bool

const (
	MODE_SEARCH_THREAD = 4
	MODE_HELP          = 3
	MODE_INPUT_THREAD  = 2
	MODE_THREAD        = 1
	MODE_BOARD         = 0
)

var clientboard *srv.Board
var activeThread *srv.Thread

var newMessage *srv.Message
var newMessageInitialText string = ""

func ClientInit() {

	clientboard = srv.CreateBoard()

	user, err := user.Current()
	if err != nil {
		panic(err)
	}
	Username = user.Username
	isAdmin = (os.Getuid() == 0)

	clientboard = FetchBoard()

	// Run de User Interface
	uiChannel = make(chan int)
	for {
		go UIRoutine(uiChannel)
		<-uiChannel
		go editorRoutine(uiChannel)
		<-uiChannel
	}
}

func editorRoutine(c chan int) {
	update := (len(newMessageInitialText) != 0)
	err, content := InputMessageFromEditor(newMessageInitialText)
	newMessageInitialText = ""
	if err == nil {
		newMessage.Text = content
		//newMessage.Save(update)
		threadKey := clientboard.Threads[boardPanel.GetThreadSelectedIndex()].Id
		if !update {
			UpdateThreadWithNewReply(newMessage, threadKey)
		} else {
			//UpdateMessage(newMessage, threadKey)
		}
		newMessage = nil
	}
	c <- 1
}

func UIRoutine(uic chan int) {
	exit := false

	DefaultStyle = tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.Color236)

	s, err := tcell.NewScreen()
	if err != nil {
		log.Fatalf("%+v", err)
	}
	if err := s.Init(); err != nil {
		log.Fatalf("%+v", err)
	}
	s.SetStyle(DefaultStyle)
	s.Clear()

	activeMode = MODE_BOARD
	confirmDelete = false

	sort.Sort(clientboard)

	boardPanel = CreateBoardPanel(s, clientboard)
	refreshPanels(s, true)

	for !exit {
		refreshPanels(s, false)
		s.Show()
		ev := s.PollEvent()
		switch ev := ev.(type) {
		case *tcell.EventResize:
			s.Sync()
			refreshPanels(s, true)

		case *tcell.EventKey:
			resetWarningMessage()
			if ev.Rune() != 'd' {
				confirmDelete = false
			}

			/*
				'ESC' key commands:
			*/
			if ev.Key() == tcell.KeyESC {
				if activeMode == MODE_BOARD {
					if clientboard.IsBoardFiltered() {
						clientboard.ResetFilter()
					} else {
						quit(s)
					}
				} else if activeMode == MODE_THREAD {
					activeMode = MODE_BOARD
				} else if activeMode == MODE_HELP {
					activeMode = lastActiveMode
				} else if activeMode == MODE_INPUT_THREAD || activeMode == MODE_SEARCH_THREAD {
					activeMode = MODE_BOARD
				}

			} else if ev.Key() == tcell.KeyDown {
				if activeMode == MODE_BOARD {
					boardPanel.DownCursor()
				}
				if activeMode == MODE_THREAD {
					threadPanel.DownCursor()
				}
			} else if ev.Key() == tcell.KeyUp {
				if activeMode == MODE_BOARD {
					boardPanel.UpCursor()
				}
				if activeMode == MODE_THREAD {
					threadPanel.UpCursor()
				}

				/*
					'Enter' key commands:
				*/
			} else if ev.Key() == tcell.KeyEnter {
				if activeMode == MODE_BOARD {
					activeThreadId := clientboard.Threads[boardPanel.GetThreadSelectedIndex()].Id
					activeThread = FetchThread(activeThreadId)
					lastActiveMode = activeMode
					activeMode = MODE_THREAD
					refreshPanels(s, true)

				} else if activeMode == MODE_INPUT_THREAD {
					title := messageBuffer.Msg
					content := ""
					newMessage = srv.NewMessage(Username, content)
					thread := srv.NewThread(title, newMessage)
					//clientboard.addThread(thread) // cambiar por API
					thread.Save()
					newMessage.Parent = thread
					exit = true // exit to run the editor
				} else if activeMode == MODE_SEARCH_THREAD {
					//pattern := messageBuffer.Msg
					//clientboard.filterThreads(pattern) // cambiar por API
					activeMode = MODE_BOARD
				}
			} else if ev.Key() == tcell.KeyPgUp {
				if activeMode == MODE_THREAD {
					threadPanel.UpPage()
				}
			} else if ev.Key() == tcell.KeyPgDn {
				if activeMode == MODE_THREAD {
					threadPanel.DownPage()
				}
			} else if ev.Key() == tcell.KeyDEL {
				messageBuffer.DelRuneFromBuffer()

			} else {
				/*
					Delete reply or new thread
				*/
				if activeMode == MODE_BOARD && ev.Rune() == 'd' {
					if !confirmDelete {
						setWarningMessage("¿Desea borrar el hilo? Pulse 'd' para confirmar o ESC para cancelar")
						confirmDelete = true
					} else {
						deleteTh := clientboard.Threads[boardPanel.GetThreadSelectedIndex()]
						if deleteTh.Author == Username {
							setWarningMessage("Borrado")
							confirmDelete = false
							//clientboard.delThread(deleteTh)
							deleteTh.Delete()
						} else {
							setWarningMessage("Solo el autor del hilo puede borrarlo")
						}
					}

				} else if activeMode == MODE_THREAD && ev.Rune() == 'd' {
					if !confirmDelete {
						setWarningMessage("¿Desea borrar la respuesta? Pulse 'd' para confirmar o ESC para cancelar")
						confirmDelete = true
					} else {
						thread := clientboard.Threads[boardPanel.GetThreadSelectedIndex()]
						deleteMsg := thread.Messages[threadPanel.MessageSelected]
						if deleteMsg.Author == Username {
							setWarningMessage("Borrado")
							confirmDelete = false
							//thread.delMessage(deleteMsg) // cambiar por API
							activeMode = MODE_BOARD
							//err := deleteMsg.Delete()
							if err != nil {
								setWarningMessage(fmt.Sprintf("%s", err))
							}
						} else {
							setWarningMessage("Solo el autor del mensaje puede borrarlo")
						}
					}

					/*
						New reply or new thread
					*/
				} else if activeMode == MODE_BOARD && ev.Rune() == 'a' {
					activeMode = MODE_INPUT_THREAD
					messageBuffer = NewMessageBuffer(s, 8)

				} else if activeMode == MODE_THREAD && ev.Rune() == 'a' {
					thread := clientboard.Threads[boardPanel.GetThreadSelectedIndex()]
					if thread.IsClosed {
						setWarningMessage("El hilo está cerrado y no admite cambios")
					} else {
						content := ""
						newMessage = srv.NewMessage(Username, content)
						//thread.addMessage(newMessage) // cambiar por API
						exit = true // exit to run the editor
					}

				} else if activeMode == MODE_BOARD && ev.Rune() == 'b' {
					/*
						Search a thread
					*/
					activeMode = MODE_SEARCH_THREAD
					messageBuffer = NewMessageBuffer(s, 10)

				} else if activeMode == MODE_THREAD && ev.Rune() == 'e' {
					/*
						Edit message
					*/
					thread := clientboard.Threads[boardPanel.GetThreadSelectedIndex()]
					if thread.IsClosed {
						setWarningMessage("El hilo está cerrado y no admite cambios")
					} else {
						newMessage = thread.Messages[threadPanel.MessageSelected]
						if newMessage.Author == Username {
							newMessageInitialText = newMessage.Text
							exit = true // exit to run the editor
						} else {
							setWarningMessage("Solo el autor del mensaje puede editarlo")
						}
					}

					/*
						Show help window
					*/
				} else if activeMode != MODE_INPUT_THREAD && ev.Rune() == '?' {
					lastActiveMode = activeMode
					activeMode = MODE_HELP

				} else if activeMode == MODE_BOARD && ev.Rune() == 'f' && isAdmin {
					/*
						Fix a thread
					*/
					thread := clientboard.Threads[boardPanel.GetThreadSelectedIndex()]
					thread.IsFixed = !thread.IsFixed
					sort.Sort(clientboard)
					thread.Update() // cambiar por API

				} else if activeMode == MODE_BOARD && ev.Rune() == 'c' && isAdmin {
					/*
						Close thread
					*/
					thread := clientboard.Threads[boardPanel.GetThreadSelectedIndex()]
					if !thread.IsClosed {
						thread.IsClosed = true // cambiar por API
						thread.Title = "[Cerrado]" + thread.Title
					} else {
						thread.IsClosed = false
						thread.Title = strings.TrimPrefix(thread.Title, "[Cerrado]")
					}
					thread.Update()

					/*
						Writting in top buffer
					*/
				} else if activeMode == MODE_INPUT_THREAD || activeMode == MODE_SEARCH_THREAD {
					messageBuffer.AddRuneToBuffer(ev.Rune())
				}
			}
		}
	}

	s.Fini()
	uic <- 1
}
