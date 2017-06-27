package gate

import (
	"net/http"
	"time"

	"fmt"

	"encoding/json"
	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 100000
)

var upgrader = websocket.Upgrader{EnableCompression: true}

type Handler func(recivedBytes []byte)

//Client provides an asynchronous transfers between the peer and the daemon
type Client struct {

	// The channel of outbound messages.
	send chan []byte

	// The websocket connection.
	conn *websocket.Conn
}

func (c *Client) SendToPeer(bytes []byte) {
	c.send <- bytes
}

// readPump pumps messages from the websocket connection to the deamon.
//
// The application runs readPump in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all
// reads from this goroutine.
func (c *Client) readPump(handler Handler) (err error) {

	for {
		messageType, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
				err = fmt.Errorf("server-client-%p readPump, UnexpectedCloseError: %v", c, err)
			}
			break
		}

		if messageType == websocket.TextMessage || messageType == websocket.BinaryMessage {
			handler(message)
		} else if messageType == websocket.CloseMessage {
			break
		}
	}
	return
}

// writePump pumps messages to the websocket connection from the deamon.
//
// A goroutine running writePump is started for each connection. The
// application ensures that there is at most one writer to a connection by
// executing all writes from this goroutine.
func (c *Client) writePump(unregister func(*Client), done chan<- struct{}, interrupt <-chan struct{}) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		unregister(c)
		close(c.send)
		done <- struct{}{}
	}()

	for {
		select {
		case <-interrupt:
			fmt.Printf("wc-client-%p interrupted!\n", c)
			return
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The server closed the channel.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				fmt.Printf("server-client-%p writePump, error getting conn.NextWriter: %v\n", c, err)
				return
			}
			w.Write(message)

			if err := w.Close(); err != nil {
				fmt.Printf("server-client-%p writePump, error closing writer: %v\n", c, err)
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				fmt.Printf("server-client-%p writePump, error closing writer: %v\n", c, err)
				return
			}
		}
	}
}

//Run handles websocket requests from the peer.
func (c *Client) Run(unregister func(*Client), handler Handler) (err error) {
	fmt.Printf("server-client-%p: start\n", c)
	done := make(chan struct{})
	interrupt := make(chan struct{}, 2)
	go c.writePump(unregister, done, interrupt)
	err = c.readPump(handler)
	interrupt <- struct{}{}
	<-done
	close(interrupt)
	close(done)
	c.conn.Close()
	fmt.Printf("server-client-%p: end\n", c)
	return
}

//NewClient сreates an object for websocket transfers to the peer.
func NewClient(w http.ResponseWriter, r *http.Request) (client *Client) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("gate.NewClient error:", err)
		return
	}

	conn.EnableWriteCompression(true)
	conn.SetReadLimit(maxMessageSize)
	conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	conn.SetCloseHandler(func(code int, text string) error {
		fmt.Printf("server%p: closed, %d, %s\n", conn, code, text)
		return nil
	})
	fmt.Printf("server%p: opened\n", conn)

	client = &Client{
		conn: conn,
		send: make(chan []byte),
	}

	return
}

func (c *Client) SendJson(object interface{}) {
	bytes, err := json.Marshal(object)
	if err != nil {
		panic(err)
	}
	c.SendToPeer(bytes)
}

func (c *Client) SendJsonError(err interface{}) {
	c.SendJson(struct {
		Error interface{}
	}{Error: err})
}
