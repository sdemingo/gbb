package client

import (
	"crypto/sha256"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path/filepath"

	"golang.org/x/term"
)

var APP_TITLE = "GBB v1.0"
var WELCOME = `

   ,o888888o.    8 888888888o   8 888888888o   
   8888      88.  8 8888     88. 8 8888     88. 
,8 8888        8. 8 8888      88 8 8888      88 
88 8888           8 8888     ,88 8 8888      88 
88 8888           8 8888.   ,88' 8 8888.    88' 
88 8888           8 8888888888   8 8888888888   
88 8888   8888888 8 8888     88. 8 8888     88. 
 8 8888       .8' 8 8888      88 8 8888      88 
   8888     ,88'  8 8888    ,88' 8 8888    ,88' 
    8888888P'    8 888888888P   8 888888888P   

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation. This program is distributed in the hope 
that it will be useful, but WITHOUT ANY WARRANTY.
`

var logFile *os.File
var logFileName = "~/gbb-debug.log"
var Username string

/*

	Client Core

	Este archivo almacena el funcionamiento inicial del cliente con el proceso de
	autenticación del usuario y el arranque de la interfaz gráfica.

	Ahora mismo la interfaz gráfica esta implementada en los archivos ui_lib y ui_core usando la librería
	tcell

*/

func readPassword() string {
	bp, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		fmt.Println("Error: no se pudo leer la contraseña del teclado")
		return ""
	}
	fmt.Println()
	return fmt.Sprintf("%s", bp)
}

func PrintWellcome(gbbdir string) {
	path := filepath.Join(gbbdir, "../stuff/wellcome")
	content, err := ioutil.ReadFile(path)
	if err != nil {
		fmt.Println(WELCOME)
	}
	fmt.Println(string(content))
}

func InitLog(enable bool) {
	if enable {
		logFile, err := os.OpenFile(logFileName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
		if err != nil {
			panic(err)
		}
		log.SetOutput(logFile)
	} else {
		log.SetOutput(ioutil.Discard)
	}
}

func logError(text string, source string) {
	if source != "" {
		log.Printf("Error in %s: %s\n", source, text)
	} else {
		log.Printf("Error: %s\n", text)
	}
}

func logEvent(text string) {
	log.Printf("%s\n", text)
}

func ClientInit(cmd string, exDir string) {

	if cmd == "--debug" {
		InitLog(true)
	} else {
		InitLog(false)
	}
	defer logFile.Close()

	user, err := user.Current()
	if err != nil {
		panic(err)
	}
	Username = user.Username

	PrintWellcome(exDir)

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

	/*
		Renew Password
	*/

	if cmd == "--password" {
		fmt.Print("Nueva contraseña: ")
		npassword1 := readPassword()
		fmt.Print("Nueva contraseña (otra vez): ")
		npassword2 := readPassword()
		if npassword1 == npassword2 {
			sum := sha256.Sum256([]byte(npassword1))
			u := RenewPassword(Username, fmt.Sprintf("%x", sum))
			if u == nil {
				fmt.Println("La contraseña no fue cambiada. Fallo la petición.")
			}
		}
		return
	}

	/*
		Reload operation request
	*/
	if cmd == "--reload" {
		err := ReloadUsers()
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Println("Se envió la petición de recarga de usuarios")
		}
	}

	/*

		Text User Interface

	*/
	if cmd == "" {
		runUI()
	}
}
