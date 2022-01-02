package main

import (
	"fmt"
	"strings"
	"time"
)

type Message struct {
	Author string
	Stamp  time.Time
	Text   string
}

func NewMessage(author string, text string) *Message {
	return &Message{Author: author, Text: text, Stamp: time.Now()}
}

func (m *Message) ResumeText() string {
	if len(m.Text) > 40 {
		return m.Text[:40]
	}
	return m.Text
}

func (m *Message) SplitInLines(nchars int) []string {
	lines := make([]string, 0)
	from := 0
	to := from + nchars
	if to >= len(m.Text) {
		to = len(m.Text)
	}
	lines = append(lines, fmt.Sprintf("De %s en %s", m.Author, m.DateString()))
	lines = append(lines, " ")
	for {
		line := m.Text[from:to]
		if strings.Contains(line, "\n") {
			sublines := strings.Split(line, "\n")
			lines = append(lines, sublines...)
		} else {
			lines = append(lines, line)
		}
		from = to
		to = from + nchars
		if to >= len(m.Text) {
			to = len(m.Text)
			line = m.Text[from:to]
			if strings.Contains(line, "\n") {
				sublines := strings.Split(line, "\n")
				lines = append(lines, sublines...)
			} else {
				lines = append(lines, line)
			}
			break
		}
	}
	lines = append(lines, " ")

	return lines
}

func (m *Message) DateString() string {
	return m.Stamp.Format(DATE_FORMAT)
}

func (m *Message) String() string {
	return fmt.Sprintf("[%s el %s] %s ...", m.Author, m.DateString(), m.ResumeText())
}

type Thread struct {
	Messages []*Message
	Title    string
}

func NewThread(title string, first *Message) *Thread {
	if first == nil {
		return nil // a thread must have a non empty first message
	}
	t := new(Thread)
	t.Messages = make([]*Message, 0)
	t.Messages = append(t.Messages, first)
	t.Title = title
	return t
}

func (t *Thread) addMessage(m *Message) {
	if m != nil {
		t.Messages = append(t.Messages, m)
	}
}

func (t *Thread) Stamp() time.Time {
	return t.Messages[0].Stamp
}

func (t *Thread) String() string {
	if len(t.Messages) > 1 {
		return fmt.Sprintf(" %s|%-10s %-2d %s ", t.Stamp().Format(DATE_FORMAT), t.Messages[0].Author, len(t.Messages)-1, t.Title)
	} else {
		return fmt.Sprintf(" %s|%-10s    %s ", t.Stamp().Format(DATE_FORMAT), t.Messages[0].Author, t.Title)
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

func (b *Board) Len() int           { return len(b.Threads) }
func (b *Board) Swap(i, j int)      { b.Threads[i], b.Threads[j] = b.Threads[j], b.Threads[i] }
func (b *Board) Less(i, j int) bool { return b.Threads[i].Stamp().After(b.Threads[j].Stamp()) }

func (b *Board) addThread(th *Thread) {
	if th != nil {
		b.Threads = append([]*Thread{th}, b.Threads...)
	}
}

func (b *Board) Print() {
	for _, th := range b.Threads {
		if th != nil {
			fmt.Printf("%s\n", th.String())
		}
	}
}
