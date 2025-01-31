package main

import (
	"fmt"
	"os"
)

func printHelp() {
	fmt.Println(`Own Redis

Usage:
  own-redis [--port <N>]
  own-redis --help

Options:
  --help       Show this screen.
  --port N     Port number.`)
	os.Exit(0)
}

func main() {
	port := "8080"
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--help":
			printHelp()
		case "--port":
			if len(os.Args) < 3 {
				fmt.Println("Ошибка: укажите порт после --port")
				os.Exit(1)
			}
			port = os.Args[2]
		default:
			fmt.Println("Неизвестный флаг:", os.Args[1])
			printHelp()
		}
	}

	if len(os.Args) > 2 && os.Args[1] == "-port" {
		port = os.Args[2]
	}

	server, err := NewServer(port)
	if err != nil {
		fmt.Println("Error running server:", err)
		os.Exit(1)
	}

	fmt.Println("Server running on port: ", port)
	server.HandleRequest()
}
