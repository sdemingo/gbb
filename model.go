package main

import (
	"fmt"
	"time"
)

var DATE_FORMAT = "02 Jan-06"

type Message struct {
	Id     int
	Parent *Thread
	Author string
	Stamp  time.Time
	Text   string
}

func NewMessage(author string, text string) *Message {
	return &Message{Parent: nil, Author: author, Text: text, Stamp: time.Now()}
}

func (m *Message) ResumeText() string {
	if len(m.Text) > 40 {
		return m.Text[:40]
	}
	return m.Text
}

func SplitStringInLines(text string, nchars int) []string {
	lines := make([]string, 0)

	count := 0
	line := ""
	for i := 0; i < len(text); i++ {
		if text[i] == '\n' {
			lines = append(lines, line)
			count = 0
			line = ""
			continue
		} else {
			line += text[i : i+1]
			count++
			if count == nchars {
				lines = append(lines, line)
				line = ""
				count = 0
			}
		}
	}
	if len(lines) > 0 {
		lines = append(lines, line)
	}
	return lines
}

func (m *Message) SplitInLines(nchars int) []string {

	msg := make([]string, 0)
	msg = append(msg, fmt.Sprintf("De %s en %s", m.Author, m.DateString()))
	msg = append(msg, " ")

	body := SplitStringInLines(m.Text, nchars)
	msg = append(msg, body...)

	msg = append(msg, " ")
	return msg
}

func (m *Message) DateString() string {
	return m.Stamp.Format(DATE_FORMAT)
}

func (m *Message) SetDate(datestr string) {
	t, err := time.Parse(DATE_FORMAT, datestr)
	if err == nil {
		m.Stamp = t
	}

}

func (m *Message) String() string {
	return fmt.Sprintf("[%s el %s] %s ...", m.Author, m.DateString(), m.ResumeText())
}

func (m *Message) Save(update bool) {
	// insert in DB
	date := m.DateString()
	q := ""
	if update {
		q = fmt.Sprintf("UPDATE messages SET content='%s' WHERE id='%d';", m.Text, m.Id)
	} else {
		q = fmt.Sprintf("INSERT INTO messages (thread, author,stamp,content) VALUES ('%s','%s','%s','%s');", m.Parent.Id, m.Author, date, m.Text)
	}
	statement, err := db.Prepare(q)

	if err == nil {
		_, err = statement.Exec()
	}
}

func (m *Message) Delete() {
	q := fmt.Sprintf("DELETE FROM messages WHERE id='%d';", m.Id)
	statement, err := db.Prepare(q)
	if err == nil {
		_, err = statement.Exec()
	}
}

type Thread struct {
	Messages []*Message
	Title    string
	Id       string
	isClosed bool
	isFixed  bool
}

func NewThread(title string, first *Message) *Thread {
	if first == nil {
		return nil // a thread must have a non empty first message
	}
	t := new(Thread)
	t.Messages = make([]*Message, 0)
	t.Messages = append(t.Messages, first)
	t.Title = title
	t.Id = RandomString(32)
	t.isClosed = false
	t.isFixed = false
	return t
}

func (t *Thread) Save() {
	// insert in DB
	closed := 0
	if t.isClosed {
		closed = 1
	}
	fixed := 0
	if t.isFixed {
		fixed = 1
	}

	q := fmt.Sprintf("INSERT INTO threads (id,title,isClosed,isFixed) VALUES ('%s','%s','%d','%d');\n", t.Id, t.Title, closed, fixed)
	statement, err := db.Prepare(q)

	if err == nil {
		_, err = statement.Exec()
	}
}

// Update
func (t *Thread) Update() {
	closed := 0
	if t.isClosed {
		closed = 1
	}
	fixed := 0
	if t.isFixed {
		fixed = 1
	}

	q := fmt.Sprintf("UPDATE threads SET isClosed='%d', isFixed='%d' WHERE id='%s';\n", closed, fixed, t.Id)
	statement, err := db.Prepare(q)

	if err == nil {
		_, err = statement.Exec()
	}
}

func (t *Thread) Delete() {
	q := fmt.Sprintf("DELETE FROM threads WHERE id='%s';", t.Id)
	statement, err := db.Prepare(q)
	if err == nil {
		_, err = statement.Exec()
	}
}

func (t *Thread) addMessage(m *Message) {
	if m != nil {
		if t.Messages == nil {
			t.Messages = make([]*Message, 0)
		}
		m.Parent = t
		t.Messages = append(t.Messages, m)
	}
}

func (t *Thread) delMessage(m *Message) {
	i := 0
	if m != nil {
		for i = range t.Messages {
			m2 := t.Messages[i]
			if (m.Author == m2.Author) && (m.Stamp == m2.Stamp) && (i > 0) {
				// El primer mensaje del hilo no se puede borrar, solo las respuestas
				break
			}
		}
		if i < len(t.Messages)-1 {
			t.Messages = append(t.Messages[:i], t.Messages[i+1:]...)
		} else {
			t.Messages = t.Messages[:len(t.Messages)-1]
		}
	}
}

func (t *Thread) CreateStamp() time.Time {
	if (t.Messages == nil) || len(t.Messages) == 0 {
		return time.Time{}
	}
	return t.Messages[0].Stamp
}

func (t *Thread) UpdateStamp() time.Time {
	if (t.Messages == nil) || len(t.Messages) == 0 {
		return time.Time{}
	}
	if len(t.Messages) > 1 {
		return t.Messages[len(t.Messages)-1].Stamp
	} else {
		return t.Messages[0].Stamp
	}
}

func (t *Thread) Author() string {
	if (t.Messages == nil) || len(t.Messages) == 0 {
		return ""
	}
	return t.Messages[0].Author
}

func (t *Thread) String() string {
	if len(t.Messages) > 1 {
		return fmt.Sprintf(" %s|%-10s %-2d %s ", t.UpdateStamp().Format(DATE_FORMAT), t.Author(), len(t.Messages)-1, t.Title)
	} else {
		return fmt.Sprintf(" %s|%-10s    %s ", t.UpdateStamp().Format(DATE_FORMAT), t.Author(), t.Title)
	}

}

type Board struct {
	Threads []*Thread
}

func CreateBoard() *Board {
	b := new(Board)
	b.Threads = make([]*Thread, 0)

	return b
}

func (b *Board) Load() error {
	// load board from database
	db := GetConnection()

	// Recuperamos todos los threads
	q := `SELECT
            id, title, isClosed, isFixed
            FROM threads`

	rows, err := db.Query(q)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var th Thread
		var closedVal, fixedVal int
		rows.Scan(
			&th.Id,
			&th.Title,
			&closedVal,
			&fixedVal,
		)
		th.isClosed = (closedVal == 1)
		th.isFixed = (fixedVal == 1)
		b.Threads = append(b.Threads, &th)
	}

	// Recuperamos todos los mensajes y los metemos en sus threads
	q = `SELECT
		id, thread, author, stamp, content
		FROM messages`

	rows, err = db.Query(q)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		m := NewMessage("", "")
		threadKey := ""
		dateString := ""
		rows.Scan(
			&m.Id,
			&threadKey,
			&m.Author,
			&dateString,
			&m.Text,
		)
		th := b.getThread(threadKey)
		if th != nil {
			m.SetDate(dateString)
			th.addMessage(m)
		}
	}

	return nil
}

func (b *Board) Len() int      { return len(b.Threads) }
func (b *Board) Swap(i, j int) { b.Threads[i], b.Threads[j] = b.Threads[j], b.Threads[i] }

func (b *Board) Less(i, j int) bool {
	if b.Threads[i].isFixed == b.Threads[j].isFixed {
		return b.Threads[i].UpdateStamp().After(b.Threads[j].UpdateStamp())
	} else {
		if b.Threads[i].isFixed {
			return true
		} else {
			return false
		}
	}

}

func (b *Board) getThread(key string) *Thread {
	for _, th := range b.Threads {
		if th.Id == key {
			return th
		}
	}
	return nil
}

func (b *Board) addThread(th *Thread) {
	if th != nil {
		b.Threads = append([]*Thread{th}, b.Threads...)
	}
}

func (b *Board) delThread(th *Thread) {
	d := 0
	if th != nil {
		for d = range b.Threads {
			if b.Threads[d].Id == th.Id {
				break
			}
		}
		if d >= 0 && d < len(b.Threads)-1 {
			b.Threads = append(b.Threads[:d], b.Threads[d+1:]...)
		} else {
			b.Threads = b.Threads[:len(b.Threads)-1]
		}
	}
}

func (b *Board) Print() {
	for _, th := range b.Threads {
		if th != nil {
			fmt.Printf("%s\n", th.String())
		}
	}
}
