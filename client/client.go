package client

import (
	"crypto/sha256"
	"fmt"
	"gbb/srv"
	"log"
	"os"
	"os/user"
	"strings"

	"github.com/gdamore/tcell"
)

/*

	Client Core

*/

func readPassword() string {
	password := ""
	fmt.Print("\033[8m") // Hide input
	fmt.Scan(&password)
	fmt.Println("\033[28m") // Show input
	return strings.Trim(password, "\n")
}

var logFile *os.File

func InitLog() {
	fileName := "client.log"
	logFile, err := os.OpenFile(fileName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		panic(err)
	}
	log.SetOutput(logFile)
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
var clientUser *srv.User
var activeThread *srv.Thread

var newMessage *srv.Message
var newMessageInitialText string = ""

var filter []string

func isBoardFiltered() bool {
	return len(filter) != 0
}

func resetFilter() {
	for i := range clientboard.Threads {
		clientboard.Threads[i].Hide = false
	}
	filter = make([]string, 0)
}

func applyFilter(filteredThreads []*srv.Thread) {
	for i := range clientboard.Threads {
		clientboard.Threads[i].Hide = true
	}
	for _, f := range filteredThreads {
		th := getThread(f.Id)
		th.Hide = false
	}
}

func getThread(key string) *srv.Thread {
	for _, th := range clientboard.Threads {
		if th.Id == key {
			return th
		}
	}
	return nil
}

func ClientInit() {
	InitLog()
	defer logFile.Close()

	user, err := user.Current()
	if err != nil {
		panic(err)
	}
	Username = user.Username

	/*

		Auth process

	*/

	clientUser = FetchUser(Username)
	if clientUser == nil {
		fmt.Println("Error: El usuario no existe. Debe solicitar un nuevo usuario")
		return
	}

	fmt.Print("Contraseña: ")
	password := readPassword()
	sum := sha256.Sum256([]byte(password))
	token := AuthUser(Username, fmt.Sprintf("%x", sum))

	if len(token) == 0 {
		fmt.Println("Error: Credenciales incorrectas")
		return
	}
	SetSessionToken(token)

	log.Printf("Login[%s] Token session:%s\n", clientUser.Login, token)

	/*

		Text User Interface

	*/

	uiChannel = make(chan int)
	for {
		go UIRoutine(uiChannel)
		<-uiChannel
		go editorRoutine(uiChannel)
		<-uiChannel
	}
}

/*



	Client Routines




*/

func editorRoutine(c chan int) {
	update := (len(newMessageInitialText) != 0)
	err, content := InputMessageFromEditor(newMessageInitialText)
	newMessageInitialText = ""
	if err == nil {
		newMessage.Text = content
		if activeThread != nil {
			if !update {
				err = UpdateThreadWithNewReply(newMessage, activeThread.Id)
			} else {
				err = UpdateContentMessage(newMessage)
			}
		}
		newMessage = nil
		if err != nil {
			setWarningMessage("Error:" + fmt.Sprintf("%s", err))
		}
	}
	c <- 1
}

func UIRoutine(uic chan int) {
	exit := false

	clientboard = FetchBoard()

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
					log.Println(filter)
					if isBoardFiltered() {
						resetFilter()
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
					if activeThread == nil {
						setWarningMessage("Error: Hilo no devuelto por el servidor")
					} else {
						lastActiveMode = activeMode
						activeMode = MODE_THREAD
					}
					refreshPanels(s, true)

				} else if activeMode == MODE_INPUT_THREAD {
					title := messageBuffer.Msg
					content := ""
					newMessage = srv.NewMessage(Username, content)
					activeThread, err = CreateThread(title)
					if err != nil {
						activeMode = MODE_BOARD
						setWarningMessage("Error: No se ha podido crear el thread")
					} else {
						exit = true // exit to run the editor and write the first message of the thread
					}

				} else if activeMode == MODE_SEARCH_THREAD {
					pattern := messageBuffer.Msg
					filter = []string{pattern}
					matches := FindThreads(pattern)
					applyFilter(matches)
					activeMode = MODE_BOARD
					refreshPanels(s, true)
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
					Delete a full thread
				*/
				if activeMode == MODE_BOARD && ev.Rune() == 'd' {
					if !confirmDelete {
						setWarningMessage("¿Desea borrar el hilo? Pulse 'd' para confirmar o ESC para cancelar")
						confirmDelete = true
					} else {
						deleteTh := clientboard.Threads[boardPanel.GetThreadSelectedIndex()]
						err := DeleteThread(deleteTh)
						if err != nil {
							setWarningMessage(fmt.Sprintf("%s", err))
						} else {
							setWarningMessage("Borrado")
							clientboard = FetchBoard()
							refreshPanels(s, true)
						}
						confirmDelete = false
					}

					/*
						Delete reply from a thread
					*/
				} else if activeMode == MODE_THREAD && ev.Rune() == 'd' {
					if !confirmDelete {
						setWarningMessage("¿Desea borrar la respuesta? Pulse 'd' para confirmar o ESC para cancelar")
						confirmDelete = true
					} else {
						thread := clientboard.Threads[boardPanel.GetThreadSelectedIndex()]
						deleteMsg := thread.Messages[threadPanel.MessageSelected]
						err := DeleteMessage(deleteMsg, thread.Id)
						if err != nil {
							setWarningMessage(fmt.Sprintf("Error: %s", err))
						} else {
							setWarningMessage("Borrado")
							clientboard = FetchBoard()
							refreshPanels(s, true)
						}
						activeMode = MODE_BOARD
						confirmDelete = false
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
						exit = true // exit to run the editor and write the first message of the thread
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
						log.Printf("--- %s --- %s \n", newMessage.Author, clientUser.Login)
						if (newMessage.Author == clientUser.Login) || clientUser.IsAdmin {
							newMessageInitialText = newMessage.Text
							exit = true // exit to run the editor and write the first message of the thread
						} else {
							setWarningMessage("Solo el autor del mensaje puede actualizarlo")
						}
					}

					/*
						Show help window
					*/
				} else if activeMode != MODE_INPUT_THREAD && ev.Rune() == '?' {
					lastActiveMode = activeMode
					activeMode = MODE_HELP

				} else if activeMode == MODE_BOARD && ev.Rune() == 'f' {
					/*
						Fix a thread
					*/
					thread := clientboard.Threads[boardPanel.GetThreadSelectedIndex()]
					if thread.IsFixed {
						err = UpdateThreadStatus(thread, "free")
					} else {
						err = UpdateThreadStatus(thread, "fixed")
					}
					if err != nil {
						setWarningMessage("Operación no permitida")
					} else {
						thread.IsFixed = !thread.IsFixed
						clientboard = FetchBoard()
						refreshPanels(s, true)
					}

				} else if activeMode == MODE_BOARD && ev.Rune() == 'c' {
					/*
						Close thread
					*/
					thread := clientboard.Threads[boardPanel.GetThreadSelectedIndex()]
					if thread.IsClosed {
						err = UpdateThreadStatus(thread, "open")
					} else {
						err = UpdateThreadStatus(thread, "close")
					}
					if err != nil {
						setWarningMessage("Operación no permitida")
					} else {
						clientboard = FetchBoard()
						refreshPanels(s, true)
					}

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
