package main

import (
	"log"
	"math/rand"
	"os/user"
	"time"

	"github.com/gdamore/tcell"
)

const HELP_TEXT = `


	Índice de teclas y comandos
	===========================

	a      -    Añade un hilo o un mensaje

	d      -    Borrar un hilo o un mensaje

	↑↓     -    Navegar entre hilos o mensajes

	AvPg   - 	Avanzar página de un mensaje

	RePg   -    Retroceder página de un mensaje

	ESC    -    Ir a la ventana anterior

	?      -    Mostrar este mensaje de ayuda
`

var DATE_FORMAT = "02 Jan-06"
var APP_TITLE = "GBB v1.0"
var DefaultStyle tcell.Style
var Username string
var activeMode = 0
var lastActiveMode = 0

const (
	MODE_HELP         = 3
	MODE_INPUT_THREAD = 2
	MODE_THREAD       = 1
	MODE_BOARD        = 0
)

var board *Board
var activeThread *Thread

var newMessage *Message
var uiChannel chan int
var confirmDelete bool

func editorRoutine(c chan int) {
	err, content := InputMessageFromEditor()
	if err == nil {
		newMessage.Text = content
		newMessage = nil
	}
	c <- 1
}

func UIRoutine(uic chan int) {

	DefaultStyle = tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.Color236)
	s, err := tcell.NewScreen()
	if err != nil {
		log.Fatalf("%+v", err)
	}
	if err := s.Init(); err != nil {
		log.Fatalf("%+v", err)
	}
	s.SetStyle(DefaultStyle)
	//s.EnablePaste()
	s.Clear()

	activeMode = MODE_BOARD
	confirmDelete = false

	boardPanel = CreateBoardPanel(s, board)
	refreshPanels(s, true)

	for {
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
			if ev.Key() == tcell.KeyESC {
				if activeMode == MODE_BOARD {
					quit(s)
				} else if activeMode == MODE_THREAD {
					activeMode = MODE_BOARD
				} else if activeMode == MODE_HELP {
					activeMode = lastActiveMode
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
			} else if ev.Key() == tcell.KeyEnter {
				if activeMode == MODE_BOARD {
					activeThread = board.Threads[boardPanel.GetThreadSelectedIndex()]
					activeMode = MODE_THREAD
					refreshPanels(s, true)
				}
				if activeMode == MODE_INPUT_THREAD {
					title := messageBuffer.Msg
					content := ""
					newMessage = NewMessage(Username, content)
					thread := NewThread(title, newMessage)
					board.addThread(thread)

					s.Fini() // destroy UI
					c := make(chan int)
					go editorRoutine(c)
					<-c
					uic <- 1 //restart UI
					break
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
						deleteTh := board.Threads[boardPanel.GetThreadSelectedIndex()]
						if deleteTh.Author() == Username {
							setWarningMessage("Borrado")
							confirmDelete = false
							board.delThread(deleteTh)
						} else {
							setWarningMessage("Solo el autor del hilo puede borrarlo")
						}
					}

				} else if activeMode == MODE_THREAD && ev.Rune() == 'd' {
					if !confirmDelete {
						setWarningMessage("¿Desea borrar la respuesta? Pulse 'd' para confirmar o ESC para cancelar")
						confirmDelete = true
					} else {
						thread := board.Threads[boardPanel.GetThreadSelectedIndex()]
						deleteMsg := thread.Messages[threadPanel.MessageSelected]
						if deleteMsg.Author == Username {
							setWarningMessage("Borrado")
							confirmDelete = false
							thread.delMessage(deleteMsg)
							activeMode = MODE_BOARD
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

					content := ""
					thread := board.Threads[boardPanel.GetThreadSelectedIndex()]
					newMessage = NewMessage(Username, content)
					thread.addMessage(newMessage)

					s.Fini() // destroy UI
					c := make(chan int)
					go editorRoutine(c)
					<-c
					uic <- 1 //restart UI
					break

					/*
						Show help window
					*/
				} else if ev.Rune() == '?' {
					lastActiveMode = activeMode
					activeMode = MODE_HELP
					/*
						Writting in top buffer
					*/
				} else if activeMode == MODE_INPUT_THREAD {
					messageBuffer.AddRuneToBuffer(ev.Rune())
				}
			}
		}
	}
}

func main() {
	rand.Seed(time.Now().UnixNano())
	user, err := user.Current()
	if err != nil {
		panic(err)
	}
	Username = user.Username
	board = createMockBoard()

	uiChannel = make(chan int)
	for {
		go UIRoutine(uiChannel)
		<-uiChannel
	}
}
