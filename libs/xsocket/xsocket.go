package xsocket

import (
	"fmt"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type XConnImpl struct {
	Conn      *websocket.Conn
	writeLock *sync.Mutex
}

type XConn interface {
	WriteJSON(v interface{}) error
	WriteMessage(messageType int, data []byte) error
	ReadMessage() (messageType int, p []byte, err error)
	Close() error
	Noop() // To avoid websocket.Conn to be embedded
}

func NewXConn(conn *websocket.Conn) XConn {
	return &XConnImpl{
		Conn:      conn,
		writeLock: &sync.Mutex{},
	}
}

func (x *XConnImpl) WriteJSON(v interface{}) error {
	x.writeLock.Lock()
	defer x.writeLock.Unlock()
	return x.Conn.WriteJSON(v)
}

func (x *XConnImpl) WriteMessage(messageType int, data []byte) error {
	x.writeLock.Lock()
	defer x.writeLock.Unlock()
	return x.Conn.WriteMessage(messageType, data)
}

func (x *XConnImpl) ReadMessage() (messageType int, p []byte, err error) {
	return x.Conn.ReadMessage()
}

func (x *XConnImpl) Close() error {
	return x.Conn.Close()
}

func (x *XConnImpl) Noop() {}

type Websocket struct {
	wsUpgrader   *websocket.Upgrader
	clients      map[XConn]bool
	mapWriteLock *sync.Mutex
	broadcast    chan interface{} // Use interface{} to support different message types
}

func NewWebsocket(wsUpgrader *websocket.Upgrader, messageType interface{}) *Websocket {
	return &Websocket{
		wsUpgrader: wsUpgrader,
		clients:    make(map[XConn]bool),
		//FIXME - This will allow all types of data to be broadcasted via the channel, we need to add checks in place for type safety
		broadcast:    make(chan interface{}),
		mapWriteLock: &sync.Mutex{},
	}
}

// UpgradeWS is a method to upgrade the HTTP connection to a WebSocket connection
func (q *Websocket) UpgradeWS(ctx *gin.Context) (*websocket.Conn, error) {
	return q.wsUpgrader.Upgrade(ctx.Writer, ctx.Request, nil)
}

// HandleMessages is a method to handle broadcasting messages to all connected clients
func (q *Websocket) BroadcastMessages() {
	for {
		message := <-q.broadcast

		q.mapWriteLock.Lock()
		clients := make([]XConn, 0, len(q.clients))
		for client := range q.clients {
			clients = append(clients, client)
		}
		q.mapWriteLock.Unlock()

		for _, client := range clients {
			err := client.WriteJSON(message)
			if err != nil {
				fmt.Printf("Error writing JSON message: %v\n", err)
				q.UnregisterClient(client)
			}
		}
	}
}

// RegisterClient is a method to register a new client connection
func (q *Websocket) RegisterClient(client XConn) {
	q.mapWriteLock.Lock()
	defer q.mapWriteLock.Unlock()
	q.clients[client] = true
}

// UnregisterClient is a method to unregister a client connection
func (q *Websocket) UnregisterClient(client XConn) {
	q.mapWriteLock.Lock()
	defer q.mapWriteLock.Unlock()
	delete(q.clients, client)
	client.Close()
}

// AddMessageToBroadcast is a method to add a message to the broadcast channel
func (q *Websocket) AddMessageToBroadcast(message interface{}) {
	q.broadcast <- message
}

// SendMessageToClient sends a message to a specific client connection
func (q *Websocket) BroadcastMessageToClient(client XConn, message interface{}) error {
	q.mapWriteLock.Lock()
	_, exists := q.clients[client]
	q.mapWriteLock.Unlock()

	if exists {
		err := client.WriteJSON(message)
		if err != nil {
			fmt.Printf("Error writing JSON message: %v\n", err)
			q.UnregisterClient(client)
			return err
		}
		return nil
	}
	return fmt.Errorf("client not found")
}
