package main

import (
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
)

func main() {
	serveMux := http.NewServeMux()

	upgrader := websocket.Upgrader{
		ReadBufferSize:  10,
		WriteBufferSize: 10,
	}

	events := make(chan Event)
	go server(events)

	serveMux.HandleFunc("/ws", func(res http.ResponseWriter, req *http.Request) {
		conn, err := upgrader.Upgrade(res, req, nil)

		if err != nil {
			fmt.Println(err)
			return
		}

		events <- &AddClient{Conn: conn}
	})

	err := http.ListenAndServe(":8000", serveMux)
	panic(err)
}

type Event interface {
	FormatEvent() string
}

type AddClient struct {
	Conn *websocket.Conn
}

func (msg *AddClient) FormatEvent() string {
	return fmt.Sprintf("AddClient %s", msg.Conn.RemoteAddr())
}

type RemoveClient struct {
	Conn *websocket.Conn
}

func (msg *RemoveClient) FormatEvent() string {
	return fmt.Sprintf("RemoveClient %s", msg.Conn.RemoteAddr())
}

type Message struct {
	SenderConn *websocket.Conn
	Message    []byte
}

func (msg *Message) FormatEvent() string {
	return fmt.Sprintf(
		"Message from %s: %s",
		msg.SenderConn.RemoteAddr(),
		string(msg.Message),
	)
}

func server(events chan Event) {
	connections := make(map[*websocket.Conn]chan []byte)

	for theEvent := range events {
		fmt.Println(theEvent.FormatEvent())

		switch e := theEvent.(type) {
		case *AddClient:
			messagesToClient := make(chan []byte, 1)

			connections[e.Conn] = messagesToClient

			go clientReader(e.Conn, events)
			go clientWriter(e.Conn, messagesToClient)

		case *RemoveClient:
			messagesToClient := connections[e.Conn]

			delete(connections, e.Conn)

			if messagesToClient != nil {
				close(messagesToClient)
			}

		case *Message:
			for conn, ch := range connections {
				if conn == e.SenderConn {
					continue
				}

				// TODO: if any client is slow to read messages, this line
				// will block the entire server
				//
				// Real chat apps have history, so the server could send the
				// special message to client asking to re-fetch missed messages
				// via history
				//
				// Another approach is to use ping-pong mechanism to detect
				// stuck connections on client, and re-fetch messages automatically
				//
				// For this chat, since client cannot read messages anyway, we
				// could drop them, but we also should somehow let the
				// client know that it skipped messages
				ch <- e.Message
			}

		default:
			panic(fmt.Sprintf("Got unknown event: %s", e))
		}
	}
}

func clientWriter(conn *websocket.Conn, messages <-chan []byte) {
	addr := conn.RemoteAddr()

	for msg := range messages {
		if err := conn.WriteMessage(websocket.BinaryMessage, msg); err != nil {
			fmt.Println(addr, "got write error ", err)
			conn.Close()
			break
		}
	}

	// Discard remaining messages. The prevents server() from blocking on
	// send
	for range messages {
	}
}

func clientReader(conn *websocket.Conn, events chan<- Event) {
	addr := conn.RemoteAddr()

	for {
		messageType, msg, err := conn.ReadMessage()

		if err != nil {
			fmt.Println(addr, "got read error ", err)

			conn.Close()
			events <- &RemoveClient{Conn: conn}

			return
		}

		if messageType != websocket.BinaryMessage {
			fmt.Println(addr, "sent text message. Expected binary")

			conn.Close()
			events <- &RemoveClient{Conn: conn}

			return
		}

		events <- &Message{Message: msg, SenderConn: conn}
	}
}
