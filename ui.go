package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"github.com/gdamore/tcell"
)

var boardPanel *BoardPanel
var threadPanel *ThreadPanel
var messageBuffer MessageBuffer
var warningMessage string
var helpMessage string

/*

	Panel representa un panel generico que se puede dibujar entre dos puntos (x1,y1) y (x2,y2)

*/

const TABSPACES = 4

type Panel struct {
	screen         tcell.Screen
	x1, y1, x2, y2 int
}

func NewPanel(scr tcell.Screen, x1, y1, x2, y2 int) *Panel {
	return &Panel{scr, x1, y1, x2, y2}
}

func (p *Panel) Resize(x1, y1, x2, y2 int) {
	p.x1 = x1
	p.y1 = y1
	p.x2 = x2
	p.y2 = y2
}

func (p *Panel) Draw() {
	// Fill background
	for row := p.y1; row <= p.y2; row++ {
		for col := p.x1; col <= p.x2; col++ {
			if row != 0 && col != 0 {
				p.screen.SetContent(col, row, ' ', nil, DefaultStyle)
			}
		}
	}

	// Draw borders
	for col := p.x1; col < p.x2-1; col++ {
		p.screen.SetContent(col, p.y1, tcell.RuneHLine, nil, DefaultStyle)
		p.screen.SetContent(col, p.y2, tcell.RuneHLine, nil, DefaultStyle)
	}

	for row := p.y1; row < p.y2; row++ {
		p.screen.SetContent(p.x1, row, tcell.RuneVLine, nil, DefaultStyle)
		p.screen.SetContent(p.x2-1, row, tcell.RuneVLine, nil, DefaultStyle)
	}

	// Only draw corners if necessary
	if p.y1 != p.y2 && p.x1 != p.x2 {
		p.screen.SetContent(p.x1, p.y1, tcell.RuneULCorner, nil, DefaultStyle)
		p.screen.SetContent(p.x2-1, p.y1, tcell.RuneURCorner, nil, DefaultStyle)
		p.screen.SetContent(p.x1, p.y2, tcell.RuneLLCorner, nil, DefaultStyle)
		p.screen.SetContent(p.x2-1, p.y2, tcell.RuneLRCorner, nil, DefaultStyle)
	}
}

// Escribe un texto entre un punto(x1,y1) y otro (x2,y2). Si el texto no cabe en esa zona
// lo corta y usa otra línea.
// Retorna el número de línea por donde se quedó escribiendo o donde terminó el mensaje
func drawText(s tcell.Screen, x1, y1, x2, y2 int, style tcell.Style, text string) int {
	row := y1
	col := x1
	for _, r := range []rune(text) {
		if r == '\t' {
			col += TABSPACES
			if col >= x2 {
				row++
				col = x1
			}
			if row > y2 {
				break
			}
		} else {
			s.SetContent(col, row, r, nil, style)
			col++
			if col >= x2 {
				row++
				col = x1
			}
			if row > y2 {
				break
			}
		}
	}

	return row
}

func quit(s tcell.Screen) {
	s.Fini()
	os.Exit(0)
}

func HelpPanel(s tcell.Screen) {
	w, h := s.Size()
	helpPanel := NewPanel(s, 0, 1, w, h-1)
	helpPanel.Draw()

	lines := SplitStringInLines(HELP_TEXT, w)
	nline := 2
	for i := range lines {
		drawText(s, 2, nline, w-1, nline, DefaultStyle, lines[i])
		nline++
	}

}

/*
	Thread Panel

	Este panel permite navegar entre los threads almacenados en una board. Permite pagina gracias a
	los parámetros:

		- CursorLine: indica la línea que esta marcando el cursor. La línea estará en un rango entre MinLine y MaxLine.
	CursorLine no señala el thread seleccionado en ese momento sino la línea.
		- FirstThreadShowed: indica el primer thread que está siendo mostrado en ese momento de entre los que
	están en el array

	Si queremos saber el thread exacto que estamos seleccionado con el cursor usaremos el método
	GetThreadSelectedIndex()

*/

type BoardPanel struct {
	Panel             *Panel
	CursorLine        int
	FirstThreadShowed int
	Board             *Board
	MaxLine           int
	MinLine           int
	MaxCol            int
}

func CreateBoardPanel(scr tcell.Screen, board *Board) *BoardPanel {
	bp := new(BoardPanel)
	bp.Board = board
	/*w, h := scr.Size()
	bp.Panel = NewPanel(scr, 0, 1, w, h-1)
	bp.FirstThreadShowed = 0
	bp.MaxLine = h - 1
	bp.MinLine = 2
	bp.MaxCol = w - 2
	bp.CursorLine = bp.MinLine*/
	bp.Init(scr)
	return bp
}

func (bp *BoardPanel) Init(scr tcell.Screen) {
	w, h := scr.Size()
	bp.Panel = NewPanel(scr, 0, 1, w, h-1)
	bp.FirstThreadShowed = 0
	bp.MaxLine = h - 1
	bp.MinLine = 2
	bp.MaxCol = w - 2
	bp.CursorLine = bp.MinLine
}

func (bp *BoardPanel) Draw() {
	bp.Panel.Draw()
	col := len(APP_TITLE) + 1
	drawText(bp.Panel.screen, 1, 0, col, 1, DefaultStyle, APP_TITLE)
	col += 10
	drawText(bp.Panel.screen, col, 0, col+20, 1, DefaultStyle, fmt.Sprintf("%d hilos", len(bp.Board.Threads)))
	col += 20
	if isAdmin {
		drawText(bp.Panel.screen, col, 0, col+20, 1, DefaultStyle, fmt.Sprintf("@%s [Admin]", Username))
	} else {
		drawText(bp.Panel.screen, col, 0, col+20, 1, DefaultStyle, fmt.Sprintf("@%s", Username))
	}

	line := 2
	for i := bp.FirstThreadShowed; i < len(bp.Board.Threads); i++ {
		if line >= bp.MaxLine {
			break
		}
		if bp.Board.Threads[i] != nil && !bp.Board.Threads[i].hide {
			text := fmt.Sprintf("%s", bp.Board.Threads[i])
			isSelected := line == bp.CursorLine
			isFixed := bp.Board.Threads[i].isFixed

			drawText(bp.Panel.screen, 1, line, bp.MaxCol, line, DefaultStyle.Reverse(isSelected).Bold(isFixed), text)
			line++
		}
	}
}

func (bp *BoardPanel) UpCursor() {
	if bp.CursorLine > bp.MinLine {
		bp.CursorLine--
	} else {
		if (bp.FirstThreadShowed - 10) >= 0 {
			bp.FirstThreadShowed = bp.FirstThreadShowed - 10
			bp.CursorLine += 10
		}
	}

}

func (bp *BoardPanel) DownCursor() {
	if bp.CursorLine < bp.MaxLine-1 && (bp.GetThreadSelectedIndex() <= len(bp.Board.Threads)) {
		bp.CursorLine++
	} else {
		if (bp.FirstThreadShowed + 10) < len(bp.Board.Threads) {
			bp.FirstThreadShowed = bp.FirstThreadShowed + 10
			bp.CursorLine -= 10
		}
	}
}

func (bp *BoardPanel) GetThreadSelectedIndex() int {
	return bp.FirstThreadShowed + bp.CursorLine - bp.MinLine
}

/*

	ThreadPanel

	Permite mostrar el contenido de un thread

*/

type ThreadPanel struct {
	Panel           *Panel
	Thread          *Thread
	MessageSelected int
	Messages        []*MessagePanel
	CursorLine      int
	FirstLineShowed int
	MaxLine         int
	MinLine         int
	MaxCol          int
}

func CreateThreadPanel(scr tcell.Screen, thread *Thread) *ThreadPanel {
	tp := new(ThreadPanel)
	w, h := scr.Size()
	tp.Thread = thread
	tp.Messages = make([]*MessagePanel, 0)
	tp.Panel = NewPanel(scr, 0, 1, w, h-1)
	tp.MaxLine = h - 2
	tp.MinLine = 2
	tp.CursorLine = tp.MinLine
	tp.FirstLineShowed = 0
	tp.MaxCol = w - 2
	tp.MessageSelected = 0

	for _, m := range tp.Thread.Messages {
		mp := CreateMessagePanel(scr, m, tp)
		tp.Messages = append(tp.Messages, mp)
	}

	return tp
}

// Este método permite dibujar un thread completo en pantalla. El método barre el array de mensajes del hilo y
// va ignorando los que quedan tras MessagesSelected.
// Tras esto comprueba si hay espacio suficiente o no para mostrar el mensaje (... if (tp.MaxLine - line) > len(mp.Lines) {...)
// 		- En caso afirmativo, se muestra completo
// 		- En caso negativo, solo mostramos la página seleccionada con mp.ActivePage
func (tp *ThreadPanel) Draw() {
	tp.CursorLine = tp.MinLine
	tp.Panel.Draw()

	drawText(tp.Panel.screen, 26, 0, tp.MaxCol, 1, DefaultStyle, tp.Thread.Title)

	line := tp.MinLine
	for indexMp, mp := range tp.Messages {
		if tp.MessageSelected > 0 {
			drawText(tp.Panel.screen, 1, 0, 25, 1, DefaultStyle, fmt.Sprintf("Respuesta %d de %d", tp.MessageSelected, len(tp.Thread.Messages)-1))
		}
		if indexMp < tp.MessageSelected {
			continue
		}
		if (tp.MaxLine - line) > len(mp.Lines) {
			// El mensaje cabe completamente
			line = mp.DrawAll(line, tp.MaxLine, (indexMp == tp.MessageSelected))
			for c := 2; c < tp.MaxCol-2; c++ {
				tp.Panel.screen.SetContent(c, line, tcell.RuneHLine, nil, DefaultStyle)
			}
			line++
		} else {
			// El mensaje no cabe completamente. Cargo la página seleccionada
			line += mp.Draw(line, tp.MaxLine, mp.ActivePage, (indexMp == tp.MessageSelected))
			break
		}
	}

}

func (tp *ThreadPanel) UpCursor() {
	if tp.MessageSelected > 0 {
		tp.MessageSelected--
	}
}

func (tp *ThreadPanel) DownCursor() {
	if tp.MessageSelected < len(tp.Messages)-1 {
		tp.MessageSelected++
	}
}

func (tp *ThreadPanel) UpPage() {
	selMes := tp.Messages[tp.MessageSelected]
	if selMes.ActivePage > 0 {
		selMes.ActivePage--
	}
}

func (tp *ThreadPanel) DownPage() {
	selMes := tp.Messages[tp.MessageSelected]
	if selMes.ActivePage < len(selMes.Pages)-1 {
		selMes.ActivePage++
	}
}

/*

	MessagePanel

	Permite mostrar el contenido de un mensaje. Los mensajes los guardamos por líneas. Las líneas se
	crearán tomando como medida la anchura del panel (w).

	Además, guardamos un array de páginas. Una página no es más que una marca de las lineas de inicio y
	fin de esta. El array de páginas nos permite saber de que línea a que línea podemos mostrar y
	no excedernos así del tamaño de la página.

	Dentro de un mensaje podremos ir navegando entre páginas usando los métodos UpPage() y DownPage()

*/

type MessagePanel struct {
	Panel      *Panel
	Parent     *ThreadPanel
	Message    *Message
	Lines      []string
	Pages      []Page
	ActivePage int
}

type Page struct {
	from int
	to   int
}

func (p Page) Len() int {
	return p.to - p.from
}

func CreateMessagePanel(scr tcell.Screen, msg *Message, parent *ThreadPanel) *MessagePanel {
	mp := new(MessagePanel)
	w, h := scr.Size()
	mp.Panel = NewPanel(scr, 0, 1, w, h-1)
	mp.Parent = parent
	mp.Message = msg
	mp.Lines = msg.SplitInLines(w - 5)
	mp.Pages = make([]Page, 0)
	mp.ActivePage = 0
	pageSize := parent.MaxLine - 2

	// Creo array de páginas
	from := 0
	to := pageSize
	if len(mp.Lines) > pageSize {
		// Mensaje de varias páginas
		for {
			if (len(mp.Lines) - from) < pageSize {
				mp.Pages = append(mp.Pages, Page{from, len(mp.Lines)})
				break
			}
			mp.Pages = append(mp.Pages, Page{from, to})

			from += pageSize
			to = from + pageSize
			if to >= len(mp.Lines) {
				to = len(mp.Lines) - 1
			}
		}
	} else {
		// Mensaje de una sola página
		mp.Pages = append(mp.Pages, Page{from, len(mp.Lines)})
	}
	return mp
}

// Este método permite dibujar un mensaje en pantalla. Comenzará a escribir en la línes startLine y
// solo mostrará la pagina marcada por npage. isSelected indicará si el mensaje es el seleccionado
// y en ese caso se resaltará su cabecera: la 1º línea de su 1º página
// Se dibujan como mucho líneas hasta endLine
// Se retornan las líneas que se han podido dibujar
func (mp *MessagePanel) Draw(startLine int, endLine int, npage int, isSelected bool) int {
	page := mp.Pages[npage]
	nline := startLine
	for i, line := range mp.Lines[page.from:page.to] {
		if nline >= endLine {
			return nline
		}
		if i == 0 && npage == 0 && isSelected {
			drawText(mp.Panel.screen, 1, nline, mp.Parent.MaxCol, nline, DefaultStyle.Reverse(true), line)
		} else if i == 0 && npage == 0 {
			drawText(mp.Panel.screen, 1, nline, mp.Parent.MaxCol, nline, DefaultStyle.Bold(true), line)
		} else {
			drawText(mp.Panel.screen, 1, nline, mp.Parent.MaxCol, nline, DefaultStyle, line)
		}
		nline++
	}
	return nline
}

// Este método permite dibujar un mensaje completo en pantalla ignorando su paginación.
// Se dibujan como mucho líneas hasta endLine
// Se retornan las líneas que se han podido dibujar
func (mp *MessagePanel) DrawAll(startLine int, endLine int, isSelected bool) int {
	nline := startLine
	for i, line := range mp.Lines {
		if nline >= (endLine - 1) {
			return nline
		}
		if i == 0 && isSelected {
			drawText(mp.Panel.screen, 1, nline, mp.Parent.MaxCol, nline, DefaultStyle.Reverse(true), line)
		} else {
			drawText(mp.Panel.screen, 1, nline, mp.Parent.MaxCol, nline, DefaultStyle, line)
		}
		nline++
	}
	return nline
}

func (mp *MessagePanel) UpPage() {
	if mp.ActivePage > 0 {
		mp.ActivePage--
	}
}

func (mp *MessagePanel) DownPage() {
	if mp.ActivePage < len(mp.Pages)-1 {
		mp.ActivePage++
	}
}

// En función del modo en el que estemos refrescará un panel u otro. Esta función ha de llamarse en el
// código principal, bien cuando queramos crear por primera vez una de las dos ventanas o bien
// cuando detectemos un redimensionado para que estas ventanas puedan actualizar sus valores de
// anchura y altura
func refreshPanels(scr tcell.Screen, resize bool) {
	scr.Clear()
	if activeMode == MODE_BOARD {
		if resize {
			boardPanel = CreateBoardPanel(scr, board)
		}
		boardPanel.Draw()
	} else if activeMode == MODE_THREAD {
		if resize {
			threadPanel = CreateThreadPanel(scr, activeThread)
		}
		threadPanel.Draw()
	} else if activeMode == MODE_INPUT_THREAD {
		InputThreadPanel(scr)
	} else if activeMode == MODE_SEARCH_THREAD {
		SearchThreadPanel(scr)
	} else if activeMode == MODE_HELP {
		HelpPanel(scr)
	}

	if board.IsBoardFiltered() {
		ShowFilteredHeader(scr)
	}

	if len(warningMessage) > 0 {
		ShowWarningMessage(scr, warningMessage)
	}

}

// Creación de la UI para crear un nuevo hilo
func InputThreadPanel(scr tcell.Screen) {
	w, _ := scr.Size()
	for col := 1; col < w; col++ {
		scr.SetContent(col, 0, ' ', nil, DefaultStyle)
	}
	drawText(scr, 1, 0, 8, 0, DefaultStyle, "Título:")
	drawText(scr, 9, 0, w, 0, DefaultStyle, messageBuffer.Msg)

	scr.ShowCursor(messageBuffer.Cursor, 0)
}

// Creación de la UI para buscar un nuevo hilo
func SearchThreadPanel(scr tcell.Screen) {
	w, _ := scr.Size()
	for col := 1; col < w; col++ {
		scr.SetContent(col, 0, ' ', nil, DefaultStyle)
	}
	drawText(scr, 1, 0, 10, 0, DefaultStyle, "Búsqueda:")
	drawText(scr, 11, 0, w, 0, DefaultStyle, messageBuffer.Msg)

	scr.ShowCursor(messageBuffer.Cursor, 0)
}

// Creación en la UI para crear un nuevo mensaje
func InputMessageFromEditor(initialText string) (error, string) {
	var body []byte

	filename := "/tmp/" + RandomString(8)

	// Write in the file the inital text
	f, err := os.Create(filename)
	if err != nil {
		return err, ""
	}
	_, err = f.WriteString(initialText)
	if err != nil {
		f.Close()
		return err, ""
	}

	cmd := exec.Command("nano", filename)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Start()
	if err != nil {
		return err, ""
	}
	err = cmd.Wait()
	if err != nil {
		return err, ""
	}
	body, err = ioutil.ReadFile(filename)
	if err != nil {
		return err, ""
	}
	content := fmt.Sprintf("%s", body)
	return nil, content
}

/*
	Buffer para manejar un texto sencillo input de una sola línea

*/

type MessageBuffer struct {
	Msg              string
	Cursor           int
	InitialCursorPos int
	screen           tcell.Screen
}

func NewMessageBuffer(scr tcell.Screen, col int) MessageBuffer {
	scr.ShowCursor(messageBuffer.Cursor, col+1)
	return MessageBuffer{"", col + 1, col, scr}
}

func (mb *MessageBuffer) AddRuneToBuffer(r rune) {
	mb.Msg = mb.Msg + string(r)
	mb.Cursor++
}

func (mb *MessageBuffer) DelRuneFromBuffer() {
	if len(mb.Msg) > 0 {
		mb.Msg = mb.Msg[:len(mb.Msg)-1]
		if mb.Cursor > mb.InitialCursorPos {
			mb.Cursor--
		}
	}
}

/*
	Mensajes de aviso en la barra superior
*/

func ShowWarningMessage(scr tcell.Screen, text string) {
	warningStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorRed)
	w, _ := scr.Size()
	for c := 1; c < w-1; c++ {
		drawText(scr, c, 0, w, 0, warningStyle, " ")
	}
	drawText(scr, 1, 0, w, 0, warningStyle, text)
}

func setWarningMessage(text string) {
	warningMessage = text
}

func resetWarningMessage() {
	warningMessage = ""
}

/*
	Cabecera informativa para las búsquedas
*/

func ShowFilteredHeader(scr tcell.Screen) {
	w, _ := scr.Size()
	for c := 1; c < w-1; c++ {
		drawText(scr, c, 0, w, 0, DefaultStyle, " ")
	}
	drawText(scr, 1, 0, w, 0, DefaultStyle, fmt.Sprintf(" Búsqueda: %s", strings.Join(board.Filter, " ")))
	scr.HideCursor()
}
