package main

import (
	"log"
	"math/rand"
	"os/user"
	"time"

	"github.com/gdamore/tcell"
)

var DATE_FORMAT = "02 Jan-06"
var APP_TITLE = "GBB Bulletin v0.1"
var DefaultStyle tcell.Style
var Username string
var activeMode = 0

const (
	MODE_INPUT_THREAD = 2
	MODE_THREAD       = 1
	MODE_BOARD        = 0
)

var board *Board
var activeThread *Thread

var boardPanel *BoardPanel
var threadPanel *ThreadPanel
var messageBuffer MessageBuffer
var newMessage *Message
var uiChannel chan int

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
	s.EnablePaste()
	s.Clear()

	activeMode = MODE_BOARD

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
			if ev.Key() == tcell.KeyCtrlC {
				quit(s)
			}
			if ev.Key() == tcell.KeyESC {
				if activeMode == MODE_BOARD {
					quit(s)
				} else if activeMode == MODE_THREAD {
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
			} else if ev.Key() == tcell.KeyEnter {
				if activeMode == MODE_BOARD {
					activeThread = board.Threads[boardPanel.GetThreadSelectedIndex()]
					activeMode = MODE_THREAD
					refreshPanels(s, true)
				}
				if activeMode == MODE_INPUT_THREAD {
					// Recogemos el título y creamos un mensaje en
					// blanco con ese titulo. En la siguiente iteración
					// del bucle no recogeremos eventos de teclado
					// hasta que nano haya terminado su trabajo
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
				if activeMode == MODE_BOARD && ev.Rune() == 'a' {
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
