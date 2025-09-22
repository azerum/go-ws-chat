package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

func main() {
	serveMux := http.NewServeMux()

	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}

	events := make(chan ServerCommand)
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

type ServerCommand interface {
	FormatServerCommand() string
}

type AddClient struct {
	Conn *websocket.Conn
}

func (msg *AddClient) FormatServerCommand() string {
	return fmt.Sprintf("AddClient %s", msg.Conn.RemoteAddr())
}

type RemoveClient struct {
	Conn *websocket.Conn
}

func (msg *RemoveClient) FormatServerCommand() string {
	return fmt.Sprintf("RemoveClient %s", msg.Conn.RemoteAddr())
}

type BroadcastMessage struct {
	SenderConn *websocket.Conn
	Message    []byte
}

func (msg *BroadcastMessage) FormatServerCommand() string {
	return fmt.Sprintf("BroadcastMessage from %s", msg.SenderConn.RemoteAddr())
}

func server(commands chan ServerCommand) {
	connToMessages := make(map[*websocket.Conn]chan []byte)

	for command := range commands {
		switch c := command.(type) {
		case *AddClient:
			log.Println(c.FormatServerCommand())

			// Small buffer is used for testing - to fill the buffer faster
			// for slow_client. Once a client has >buffer pending messages,
			// it will be disconnected. See broadcasting code below
			messages := make(chan []byte, 5)

			connToMessages[c.Conn] = messages

			go clientReader(c.Conn, commands)
			go clientWriter(c.Conn, messages)

		case *RemoveClient:
			log.Println(c.FormatServerCommand())

			messages := connToMessages[c.Conn]

			if messages != nil {
				close(messages)
			}

			delete(connToMessages, c.Conn)

		case *BroadcastMessage:
			connectionsToClose := make([]*websocket.Conn, 0)

			for connection, ch := range connToMessages {
				if connection == c.SenderConn {
					continue
				}

				// Avoid blocking sends, so a single slow-to-read client
				// won't block the rest of the chat
				//
				// Buffer some messages for temporary slowness of the client.
				// If it continues to not read messages in time, disconnect it
				select {
				case ch <- c.Message:
				default:
					log.Println(connection.RemoteAddr(), "is too slow")
					connectionsToClose = append(connectionsToClose, connection)
				}
			}

			for _, conn := range connectionsToClose {
				messages := connToMessages[conn]

				if messages != nil {
					close(messages)
				}

				delete(connToMessages, conn)

				// Unblock reader/writer. Reader will issue `RemoveClient`
				// command which will be a noop, as the code above has already
				// removed the client
				conn.Close()
			}

		default:
			panic(fmt.Sprintf("Got unknown event: %s", c))
		}
	}
}

func clientWriter(conn *websocket.Conn, messages <-chan []byte) {
	addr := conn.RemoteAddr()

	for msg := range messages {
		if err := conn.WriteMessage(websocket.BinaryMessage, msg); err != nil {
			log.Println(addr, "got write error ", err)
			conn.Close()
			break
		}
	}
}

func clientReader(conn *websocket.Conn, commands chan<- ServerCommand) {
	addr := conn.RemoteAddr()

	for {
		messageType, msg, err := conn.ReadMessage()

		if err != nil {
			log.Println(addr, "got read error ", err)

			conn.Close()
			commands <- &RemoveClient{Conn: conn}

			return
		}

		if messageType != websocket.BinaryMessage {
			fmt.Println(addr, "sent text message. Expected binary")

			conn.Close()
			commands <- &RemoveClient{Conn: conn}

			return
		}

		commands <- &BroadcastMessage{Message: msg, SenderConn: conn}
	}
}
