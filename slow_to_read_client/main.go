package main

import (
	"log"
	"time"

	"github.com/gorilla/websocket"
)

func main() {
	conn, _, err := websocket.DefaultDialer.Dial("ws://localhost:8000/ws", nil)

	if err != nil {
		panic(err)
	}

	sleepDuration := 10 * time.Second

	for {
		_, msg, err := conn.ReadMessage()

		if err != nil {
			log.Fatal(err)
		}

		log.Printf("Got %s. Sleeping %s", (string)(msg), sleepDuration)
		time.Sleep(sleepDuration)
	}
}
