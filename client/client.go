package client

import (
	"crypto/sha256"
	"fmt"
	"log"
	"os"
	"os/user"
	"strings"
)

var APP_TITLE = "GBB v0.1"

var logFile *os.File
var logFileName = "/tmp/client.log"

var Username string
var isAdmin bool

/*

	Client Core

	Este archivo almacena el funcionamiento inicial del cliente con el proceso de
	autenticación del usuario y el arranque de la interfaz gráfica.

	Ahora mismo la interfaz gráfica esta implementada en los archivos ui_lib y ui_core usando la librería
	tcell

*/

func readPassword() string {
	password := ""
	fmt.Print("\033[8m") // Hide input
	fmt.Scan(&password)
	fmt.Println("\033[28m") // Show input
	return strings.Trim(password, "\n")
}

func InitLog() {
	logFile, err := os.OpenFile(logFileName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		panic(err)
	}
	log.SetOutput(logFile)
}

func ClientInit() {
	InitLog()
	defer logFile.Close()

	user, err := user.Current()
	if err != nil {
		panic(err)
	}
	Username = user.Username

	/*

		Auth process

	*/

	clientUser = FetchUser(Username)
	if clientUser == nil {
		fmt.Println("Error: El usuario no existe. Debe solicitar un nuevo usuario")
		return
	}

	fmt.Print("Contraseña: ")
	password := readPassword()
	sum := sha256.Sum256([]byte(password))
	token := AuthUser(Username, fmt.Sprintf("%x", sum))

	if len(token) == 0 {
		fmt.Println("Error: Credenciales incorrectas")
		return
	}
	SetSessionToken(token)

	log.Printf("Login[%s] Token session:%s\n", clientUser.Login, token)

	/*

		Text User Interface

	*/

	runUI()
}
