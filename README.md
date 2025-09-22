Simple WebSockets chat in Go: when client sends a message, it is broadcasted
to everybody else. Does not persist any history

## Usage

Start server:

```
go run .
```

Start client(s):

```
go run ./client
```

Type messages to stdin, see messages in stdout

## Share memory by communicating

Instead of maintaining a set of connected clients and accessing it concurrently 
(with locks), the set of clients is stored only in `server()` goroutine

Clients are added/removed by sending `AddClient`/`RemoveClient` commands to 
the `server()`, which processes them one-by-one, hence no concurrency issues

## Crash-only

Shutdown code is not implemented on client nor on server. Rationale is that
default Ctrl+C works good enough: if client crashes, robust server must cleanup
broken connection anyway

## Protection from slow-to-read clients

What to do when chat continues to send messages, but one client does not 
read them? 

Silently dropping them is a bad UX, as users won't be aware
of messages missing

Server can buffer messages (and should, to deliver messages on occasional lags),
but buffer must be bounded to prevent DoS. What to do once buffer is full?

A simple solution is to disconnect the client completely, as they are likely
having bad connection and won't be able to read messages anyway. With large
enough buffer, this should not affect most users

#### Observing disconnection

`./slow_to_read_client` is a client that reads one message per 10s. You
can start a chatty client to overwhelm the slow-reading one and see how
it will get disconnected:

1. Start server

```shell
go run .
```

2. Start slow client

```shell
go run ./slow_to_read_client
```

3. Start chatty client

```shell
yes | go run ./client
```

Output of the server should be similar to:

```
2025/09/22 12:47:35 AddClient [::1]:50211
2025/09/22 12:47:36 AddClient [::1]:50214
[::1]:50211 is too slow
2025/09/22 12:47:36 [::1]:50211 got read error  read tcp [::1]:8000->[::1]:50211: use of closed network connection
2025/09/22 12:47:36 [::1]:50211 got write error  write tcp [::1]:8000->[::1]:50211: use of closed network connection
2025/09/22 12:47:36 RemoveClient [::1]:50211
```

> In this example, `[::1]:50211` is the slow client

Output of the slow client should be similar to:

```
2025/09/22 12:42:42 Got y. Sleeping 10s
2025/09/22 12:42:52 Got y. Sleeping 10s
2025/09/22 12:43:02 Got y. Sleeping 10s
2025/09/22 12:43:12 Got y. Sleeping 10s
2025/09/22 12:43:22 Got y. Sleeping 10s
2025/09/22 12:43:32 websocket: close <some reason>
```
