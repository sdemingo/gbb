package main

import (
	"gbb/client"
	"gbb/srv"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func GetGBBBinDirectory() string {
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	exPath := filepath.Dir(ex)
	return exPath
}

func main() {
	rand.Seed(time.Now().UnixNano())

	exDir := GetGBBBinDirectory()

	if len(os.Args) > 1 && os.Args[1] == "--server" {
		//Run server mode:
		srv.ServerInit(exDir)

	} else {
		//Run in client mode:
		if len(os.Args) > 1 {
			client.ClientInit(os.Args[1], exDir)
		} else {
			client.ClientInit("", exDir)
		}
	}
}
