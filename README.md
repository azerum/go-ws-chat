Simple WebSockets chat in Go: when client sends a message, it is broadcasted
to everybody else. Persists no history

### Usage

Start server:

```
cd server
go run main.go
```

Start client(s):

```
cd client
go run main.go
```

Type messages to stdin, see messages in stdout

### Share memory by communicating

Instead of maintaining a set of connected clients and accessing it concurrently 
(with locks), the set of clients is stored only in `server()` goroutine

Clients are added/removed by sending `AddClient`/`RemoveClient` messages to 
the `server()`, which processes them one-by-one, hence no concurrency issues

### Crash-only

Shutdown code is not implemented on client nor on server. Rationale is that
default Ctrl+C works good enough: if client crashes, robust server must cleanup
broken connection anyway
