package main

import (
	"bufio"
	"fmt"
	"os"

	"github.com/gorilla/websocket"
)

func main() {
	conn, _, err := websocket.DefaultDialer.Dial("ws://localhost:8000/ws", nil)

	if err != nil {
		panic(err)
	}

	go reader(conn)
	writer(conn)
}

func reader(conn *websocket.Conn) {
	defer conn.Close()

	for {
		messageType, message, err := conn.ReadMessage()

		if err != nil {
			fmt.Printf("Read error: %s", err)
			return
		}

		if messageType != websocket.BinaryMessage {
			fmt.Printf("Read text message. Expected binary")
			return
		}

		text := string(message)
		fmt.Println(text)
	}
}

func writer(conn *websocket.Conn) {
	defer conn.Close()
	scanner := bufio.NewScanner(os.Stdin)

	for scanner.Scan() {
		message := scanner.Bytes()

		err := conn.WriteMessage(websocket.BinaryMessage, message)

		if err != nil {
			fmt.Printf("Write error: %s", err)
			return
		}
	}
}
