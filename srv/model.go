package srv

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"
)

var DATE_FORMAT = "02/01/06"

/*

	Mensajes

*/

type Message struct {
	Id     int       `json:"id"`
	Parent *Thread   `json:"-"`
	Author string    `json:"author"`
	Stamp  time.Time `json:"stamp"`
	Text   string    `json:"text"`
}

func NewMessage(author string, text string) *Message {
	return &Message{Parent: nil, Author: author, Text: text, Stamp: time.Now()}
}

// Trocea el texto en líneas de, como máximo nchars.
// Respeta el word wrapping
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
				if !strings.Contains(line, " ") && !strings.Contains(line, ",.;:()[]") {
					break
				}
				nline := strings.TrimRightFunc(line, func(r rune) bool {
					return !unicode.IsSpace(r) && !unicode.IsPunct(r)
					/*
						nline será un string desde el último caracter procesado hasta
						el anterior espacio o signo de puntuación
					*/
				})
				i -= (len(line) - len(nline)) // retraso i la diferencia entre line y nline
				line = nline
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

// Función que utiliza a SplitStringInLines para separar en varías líneas
// el contenido de un mensaje
func (m *Message) SplitInLines(nchars int) []string {
	msg := make([]string, 0)
	msg = append(msg, " ")
	body := SplitStringInLines(m.Text, nchars)
	msg = append(msg, body...)

	msg = append(msg, " ")
	return msg
}

// Imprime la fecha en el formato establecido
func (m *Message) DateString() string {
	return m.Stamp.Format(DATE_FORMAT)
}

// Fija una fecha nueva a un mensaje
func (m *Message) SetDate(datestr string) {
	t, err := time.Parse(DATE_FORMAT, datestr)
	if err == nil {
		m.Stamp = t
	}
}

// Inserta un mensaje nuevo o actualiza un mensaje ya guardado en la base de datos.
// El parámetro update decide esto. En el caso de ser actualizado, solo podemos
// cambiar el contenido del mensaje
func (m *Message) Save(update bool) error {
	date := m.DateString()

	escapeText:=strings.Replace(m.Text,"'","''",-1)
	q := ""
	if update {
		q = fmt.Sprintf("UPDATE messages SET content='%s' WHERE id='%d';", escapeText, m.Id)
	} else {
		q = fmt.Sprintf("INSERT INTO messages (thread, author,stamp,content) VALUES ('%s','%s','%s','%s');", m.Parent.Id, m.Author, date, escapeText)
	}

	db,err:=GetConnection()
	defer CloseConnection(db)
	if err!=nil{return err}

	statement, err := db.Prepare(q)
	if err != nil {
		return err
	}

	res, err := statement.Exec()
	if err != nil {
		return err
	}
	if !update {
		id, err := res.LastInsertId()
		if err != nil {
			return err
		}
		m.Id = int(id)
	}
	return nil
}

// Borra un mensaje de la base de datos
func (m *Message) DeleteFromBD() error {
	q := fmt.Sprintf("DELETE FROM messages WHERE id=%d;", m.Id)

	db,err:=GetConnection()
	defer CloseConnection(db)
	if err!=nil{return err}
	statement, err := db.Prepare(q)
	if err != nil {
		return err
	}
	_, err = statement.Exec()
	
	return err
}

func (m *Message) String() string {
	return fmt.Sprintf("[%d][%s][%s] %s ", m.Id, m.Stamp.Format(DATE_FORMAT), m.Author, m.Text)
}

/*

	Hilos

*/

type Thread struct {
	Messages    []*Message
	Id          string    `json:"id"`
	Len         int       `json:"len"`
	Author      string    `json:"author"`
	UpdateStamp time.Time `json:"ustamp"`
	CreateStamp time.Time `json:"cstamp"`
	Title       string    `json:"title"`
	IsClosed    bool      `json:"isclosed"`
	IsFixed     bool      `json:"isfixed"`
	Hide        bool
}

func NewThread(title string, first *Message) *Thread {
	t := new(Thread)
	t.Messages = make([]*Message, 0)
	if first != nil {
		t.Messages = append(t.Messages, first)
	}
	t.Title = title
	t.Len = 0
	t.Id = RandomString(32)
	t.IsClosed = false
	t.IsFixed = false
	t.Hide = false
	return t
}

// Inserta un hilo nuevo en la base de datos.
func (t *Thread) Save() error {
	closed := 0
	if t.IsClosed {
		closed = 1
	}
	fixed := 0
	if t.IsFixed {
		fixed = 1
	}
	q := fmt.Sprintf("INSERT INTO threads (id,title,IsClosed,IsFixed) VALUES ('%s','%s','%d','%d');\n", t.Id, t.Title, closed, fixed)

	db,err:=GetConnection()
	defer CloseConnection(db)
	if err!=nil{return err}

	statement, err := db.Prepare(q)
	if err != nil {
		return err
	}
	_, err = statement.Exec()
	return err
	
}

// Actualiza un hilo de la base de datos. Solo pueden ser actualizados los campos de fixed o closed.
func (t *Thread) Update() error{
	closed := 0
	if t.IsClosed {
		closed = 1
	}
	fixed := 0
	if t.IsFixed {
		fixed = 1
	}

	q := fmt.Sprintf("UPDATE threads SET IsClosed='%d', IsFixed='%d' WHERE id='%s';\n", closed, fixed, t.Id)

	db,err:=GetConnection()
	defer CloseConnection(db)
	if err!=nil{return err}

	statement, err := db.Prepare(q)
	if err != nil {
		return err
	}
	_, err = statement.Exec()
	return err
}

// Borra un hilo de la base de datos
func (t *Thread) Delete() error {
	for _, m := range t.Messages {
		m.DeleteFromBD()
	}
	q := fmt.Sprintf("DELETE FROM threads WHERE id='%s';", t.Id)

	db,err:=GetConnection()
	defer CloseConnection(db)
	if err!=nil{return err}

	statement, err := db.Prepare(q)
	if err != nil {
		return err
	}
	_, err = statement.Exec()
	return err
}

// Añade un mensaje al hilo
func (t *Thread) addMessage(m *Message) {
	if m != nil {
		if t.Messages == nil {
			t.Messages = make([]*Message, 0)
		}
		m.Parent = t
		t.Messages = append(t.Messages, m)
	}
	t.Len = len(t.Messages)
	t.UpdateStamp = t.GetUpdateStamp()
	t.CreateStamp = t.GetCreateStamp()
	t.Author = t.GetAuthor()
}

// Elimina un mensaje del hilo
func (t *Thread) delMessage(m *Message) error {
	i := 0
	found := false
	if m != nil {
		for i = range t.Messages {
			m2 := t.Messages[i]
			//if (m.Author == m2.Author) && (m.Stamp == m2.Stamp) && (i > 0) {
			if (m.Id == m2.Id) && (i > 0) {
				// El primer mensaje del hilo no se puede borrar, solo las respuestas
				found = true
				break
			}
		}
		if !found {
			return errors.New("El mensaje buscado no existe")
		}
		if i < len(t.Messages)-1 {
			t.Messages = append(t.Messages[:i], t.Messages[i+1:]...)
		} else {
			t.Messages = t.Messages[:len(t.Messages)-1]
		}
	}
	t.Len = len(t.Messages)
	return nil
}

// Retorna la fecha de creación del hilo (la del primer mensaje)
func (t *Thread) GetCreateStamp() time.Time {
	if (t.Messages == nil) || len(t.Messages) == 0 {
		return time.Time{}
	}
	return t.Messages[0].Stamp
}

// Retorna la fecha de modificación del hilo (la del último mensaje)
func (t *Thread) GetUpdateStamp() time.Time {
	if (t.Messages == nil) || len(t.Messages) == 0 {
		return time.Time{}
	}
	if len(t.Messages) > 1 {
		return t.Messages[len(t.Messages)-1].Stamp
	} else {
		return t.Messages[0].Stamp
	}
}

// Retorna el autor del hilo (el del primer mensaje)
func (t *Thread) GetAuthor() string {
	if (t.Messages == nil) || len(t.Messages) == 0 {
		return ""
	}
	return t.Messages[0].Author
}

func (t *Thread) String() string {
	if t.IsClosed {
		return fmt.Sprintf(" %s|%-20s !! %s ", t.UpdateStamp.Format(DATE_FORMAT), t.Author, t.Title)
	}
	if (t.Len) > 1 {
		return fmt.Sprintf(" %s|%-20s %-2d %s ", t.UpdateStamp.Format(DATE_FORMAT), t.Author, t.Len-1, t.Title)
	} else {
		return fmt.Sprintf(" %s|%-20s    %s ", t.UpdateStamp.Format(DATE_FORMAT), t.Author, t.Title)
	}
}

/*

	El tablón

*/

type Board struct {
	Threads []*Thread
	//Filter  []string `json:"-"`
	Users []*User `json:"-"`
}

func CreateBoard() *Board {
	b := new(Board)
	b.Threads = make([]*Thread, 0)
	b.Users = make([]*User, 0)
	//b.Filter = make([]string, 0)
	return b
}

// Carga de la base de datos solo la tabla de usuarios
func (b *Board) LoadUsers() error {
	q := `SELECT login, password, isAdmin, isBanned FROM users`

	db,err:=GetConnection()
	defer CloseConnection(db)
	if err!=nil{return err}

	rows, err := db.Query(q)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		login := ""
		pass := make([]byte, 100)
		isBanned := 0
		isAdmin := 0
		rows.Scan(
			&login,
			&pass,
			&isAdmin,
			&isBanned,
		)

		u := NewUser(login, pass)
		u.IsAdmin = (isAdmin == 1)
		u.IsBanned = (isBanned == 1)
		b.AddUser(u)
	}

	return nil
}

// Carga toda la base de datos
func (b *Board) Load() error {
	db,err := GetConnection()
	//defer CloseConnection(db)
	if err!=nil{
		CloseConnection(db)
		return err	
	}

	// Recuperamos los threads
	q := `SELECT
            id, title, IsClosed, IsFixed
            FROM threads`

	rows, err := db.Query(q)
	if err != nil {
		CloseConnection(db)
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
		th.IsClosed = (closedVal == 1)
		th.IsFixed = (fixedVal == 1)
		th.Hide = false
		b.Threads = append(b.Threads, &th)
	}

	// Recuperamos todos los mensajes y los metemos en sus threads
	q = `SELECT
		id, thread, author, stamp, content
		FROM messages`

	rows, err = db.Query(q)
	if err != nil {
		CloseConnection(db)
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

	CloseConnection(db)

	err = b.LoadUsers()

	return err
}

func (b *Board) Len() int      { return len(b.Threads) }
func (b *Board) Swap(i, j int) { b.Threads[i], b.Threads[j] = b.Threads[j], b.Threads[i] }
func (b *Board) Less(i, j int) bool {
	if b.Threads[i].IsFixed == b.Threads[j].IsFixed {
		return b.Threads[i].UpdateStamp.After(b.Threads[j].UpdateStamp)
	} else {
		if b.Threads[i].IsFixed {
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

func (b *Board) getMessage(id int) *Message {
	for _, th := range b.Threads {
		for _, m := range th.Messages {
			if m.Id == id {
				return m
			}
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

// Filter board's threads. Return an array of threads that
// contain the pattern
func (b *Board) filterThreads(filter string) []*Thread {
	patterns := []string{filter}
	matched := make([]*Thread, 0)
	appended := make(map[string]int)
	for i := range b.Threads {
		for _, m := range b.Threads[i].Messages {
			for _, w := range patterns {
				if strings.Index(m.Text, w) >= 0 {
					if _, ok := appended[b.Threads[i].Id]; !ok {
						// if the thread nos found yet, it is appended to matched
						matched = append(matched, b.Threads[i])
						appended[b.Threads[i].Id] = 1
					}
				}
			}
		}
	}
	return matched
}

func (b *Board) AddUser(u *User) {
	b.Users = append(b.Users, u)
}

func (b *Board) GetUser(login string) *User {
	for _, u := range b.Users {
		if u.Login == login {
			return u
		}
	}
	return nil
}

/*

	Usuarios

*/

type User struct {
	Login    string `json:"login"`
	Password []byte `json:"-"`
	IsAdmin  bool   `json:"isadmin"`
	IsBanned bool   `json:"isbanned"`
}

func NewUser(login string, pass []byte) *User {
	u := new(User)
	u.Login = login
	u.Password = pass
	u.IsAdmin = false
	u.IsBanned = false
	return u
}

// Save or update an user in the database
func (u *User) Save(update bool) error{
	q := ""
	isadmin := 0
	if u.IsAdmin {
		isadmin = 1
	}
	isbanned := 0
	if u.IsBanned {
		isbanned = 1
	}
	password := string(u.Password[:])
	if update {
		q = fmt.Sprintf("UPDATE users SET password='%s',isAdmin='%d', isBanned='%d' WHERE login='%s';", password, isadmin, isbanned, u.Login)
	} else {
		q = fmt.Sprintf("INSERT INTO users (login,password,isAdmin,isBanned) VALUES ('%s','%s','0','0');", u.Login, password)
	}

	db,err:=GetConnection()
	defer CloseConnection(db)
	if err!=nil{return err}

	statement, err := db.Prepare(q)
	if err != nil {
		return err
	}
	_, err = statement.Exec()
	return err
}
