package main

import (
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"time"
)

/*

	Mock Messages

	For testing and debug

*/

var words = []string{
	"ad",
	"adipisicing",
	"aliqua",
	"aliquip",
	"amet",
	"anim",
	"aute",
	"cillum",
	"commodo",
	"consectetur",
	"consequat",
	"culpa",
	"cupidatat",
	"deserunt",
	"do",
	"dolor",
	"dolore",
	"duis",
	"ea",
	"eiusmod",
	"elit",
	"enim",
	"esse",
	"est",
	"et",
	"eu",
	"ex",
	"excepteur",
	"exercitation",
	"fugiat",
	"id",
	"in",
	"incididunt",
	"ipsum",
	"irure",
	"labore",
	"laboris",
	"laborum",
	"Lorem",
	"magna",
	"minim",
	"mollit",
	"nisi",
	"non",
	"nostrud",
	"nulla",
	"occaecat",
	"officia",
	"pariatur",
	"proident",
	"qui",
	"quis",
	"reprehenderit",
	"sint",
	"sit",
	"sunt",
	"tempor",
	"ullamco",
	"ut",
	"velit",
	"veniam",
	"voluptate",
}

var names = []string{"sergio",
	"sdemingo",
	/*"luiskan",
	"fterror",
	"arkainoso",
	"fefeiro",
	"apolut",
	"karo",*/
	"migualer"}

var MIN_WORDS_PER_MESSAGE = 100
var MAX_WORDS_PER_MESSAGE = 1000

var MAX_BOARD_NUM_THREADS = 150
var MIN_BOARD_NUM_THREADS = 80

var MAX_MESSAGES_PER_THREAD = 5
var MIN_MESSAGES_PER_THREAD = 0

func RandomString(len int) string {
	b := make([]byte, len)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}

func RandomText(min, max int) string {
	text := ""
	if min >= max {
		min = 0
	}
	nwords := rand.Intn(max-min) + min
	for i := 0; i < nwords; i++ {
		w := words[rand.Intn(len(words)-1)]
		text = text + " " + w
	}

	return strings.Title(text)
}

func RandomThread() *Thread {
	first := RandomMessage()
	title := RandomText(5, 10)
	th := NewThread(title, first)

	return th
}

func RandomDate() time.Time {
	min := time.Date(2016, 1, 0, 0, 0, 0, 0, time.UTC).Unix()
	max := time.Date(2021, 1, 0, 0, 0, 0, 0, time.UTC).Unix()
	delta := max - min

	sec := rand.Int63n(delta) + min
	return time.Unix(sec, 0)
}

func RandomMessage() *Message {
	author := names[rand.Intn(len(names)-1)]
	text := RandomText(MIN_WORDS_PER_MESSAGE, MAX_WORDS_PER_MESSAGE)
	text = text + "[fin]"
	msg := NewMessage(author, text)
	msg.Stamp = RandomDate()
	return msg
}

func createMockBoard() *Board {
	b := CreateBoard()

	nthreads := rand.Intn(MAX_BOARD_NUM_THREADS-MIN_BOARD_NUM_THREADS) + MIN_BOARD_NUM_THREADS
	for i := 0; i < nthreads; i++ {
		th := RandomThread()
		b.addThread(th)
		nmessages := rand.Intn(MAX_MESSAGES_PER_THREAD-MIN_MESSAGES_PER_THREAD) + MIN_MESSAGES_PER_THREAD
		for i := 0; i < nmessages; i++ {
			m := RandomMessage()
			th.addMessage(m)
		}
	}

	sort.Sort(b)
	return b
}
