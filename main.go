package main

import (
	"gbb/client"
	"gbb/srv"
	"math/rand"
	"os"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	if len(os.Args) > 1 && os.Args[1] == "--server" {
		//Run server mode:
		srv.ServerInit()

	} else {
		//Run in client mode:
		if len(os.Args) > 1 {
			client.ClientInit(os.Args[1])
		} else {
			client.ClientInit("")
		}
	}
}
