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
	MODE_INPUT_TITLE = 2
	MODE_THREAD      = 1
	MODE_BOARD       = 0
)

var board *Board
var activeThread *Thread

var boardPanel *BoardPanel
var threadPanel *ThreadPanel
var messageBuffer MessageBuffer

func main() {
	rand.Seed(time.Now().UnixNano())
	user, err := user.Current()
	if err != nil {
		panic(err)
	}
	Username = user.Username
	board = createMockBoard()

	DefaultStyle = tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.Color236)
	s, err := tcell.NewScreen()
	if err != nil {
		log.Fatalf("%+v", err)
	}
	if err := s.Init(); err != nil {
		log.Fatalf("%+v", err)
	}
	s.SetStyle(DefaultStyle)
	s.EnableMouse()
	s.EnablePaste()
	s.Clear()

	boardPanel = CreateBoardPanel(s, board)

	for {
		refreshPanels(s, false)
		s.Show()

		// Process event
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
				if activeMode == MODE_INPUT_TITLE {
					// recogemos el título escrito
					// y creamos el nuevo thread
					title := messageBuffer.Msg
					content := "contenido del nuevo mensaje"
					newm := NewMessage(Username, content)
					thread := NewThread(title, newm)
					board.addThread(thread)
					activeMode = MODE_BOARD
					refreshPanels(s, true) //?¿?¿?
					s.HideCursor()
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
					Pulso caracteres normales
				*/
				if activeMode == MODE_BOARD && ev.Rune() == 'a' {
					activeMode = MODE_INPUT_TITLE
					messageBuffer = NewMessageBuffer(s, 8)
				} else if activeMode == MODE_INPUT_TITLE {
					messageBuffer.AddRuneToBuffer(ev.Rune())
				}
			}
		}
	}
}
