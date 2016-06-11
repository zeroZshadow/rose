package rose

import (
	"net"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 4096
	maxMessages    = 8
)

// MessageType message id type
type MessageType uint64

// MessagePump pump used for reading and writing on a websocket
type MessagePump struct {
	server *Server
	user   User

	Connected bool
	Address   net.Addr

	// The websocket connection.
	ws *websocket.Conn

	// Buffered channel of outbound messages.
	outgoing chan []byte
	lock     *sync.RWMutex
}

// NewMessagePump create a new MessagePump
func NewMessagePump(constructor UserConstructor, ws *websocket.Conn, server *Server) (pump *MessagePump) {
	// Create new pump
	pump = &MessagePump{
		server:   server,
		ws:       ws,
		outgoing: make(chan []byte, maxMessages),
		lock:     new(sync.RWMutex),
	}

	// Create new user based on pattern
	user := constructor(pump)
	pump.user = user
	pump.Connected = true
	pump.Address = ws.RemoteAddr()

	user.OnConnect()

	return
}

// readPump pumps messages from the websocket connection.
func (pump *MessagePump) readPump() {
	// Setup read pump
	pump.ws.SetReadLimit(maxMessageSize)
	pump.ws.SetReadDeadline(time.Now().Add(pongWait))
	pump.ws.SetPongHandler(func(string) error {
		return pump.ws.SetReadDeadline(time.Now().Add(pongWait))
	})

	// Loop to read messages until we need to exit
	for {
		// Read a message of the websocket, blocking
		_, data, err := pump.ws.ReadMessage()
		if err != nil {
			break
		}

		// Read message type
		messageType, bytes := proto.DecodeVarint(data)
		message := data[bytes:]

		// Send packet to user code
		pump.user.HandlePacket(MessageType(messageType), message)
	}

	// The lock is so we're sure nothing writes to the pump while we close things
	pump.lock.Lock()
	// Close the writing channel so the rest stops
	if pump.Connected {
		close(pump.outgoing)
	}
	// Set user as disconnected
	pump.Connected = false
	pump.lock.Unlock()

	pump.user.OnDisconnect(nil)
}

// write writes a message with the given message type and payload.
func (pump *MessagePump) writeRaw(mt int, payload []byte) error {
	pump.ws.SetWriteDeadline(time.Now().Add(writeWait))
	return pump.ws.WriteMessage(mt, payload)
}

// writePump pumps messages from the hub to the websocket connection.
func (pump *MessagePump) writePump() {
	pingTicker := time.NewTicker(pingPeriod)

writeloop:
	for {
		select {
		case message, ok := <-pump.outgoing:
			// Did the channel close? send close message over socket!
			if !ok || message == nil {
				pump.writeRaw(websocket.CloseMessage, []byte{})
				break writeloop
			}
			// Write message to socket
			if err := pump.writeRaw(websocket.BinaryMessage, message); err != nil {
				break writeloop
			}
		case <-pingTicker.C:
			// Send ping message to keep connection alive
			if err := pump.writeRaw(websocket.PingMessage, []byte{}); err != nil {
				break writeloop
			}
		}
	}

	// The lock is so we're sure nothing writes to the pump while we close things
	pump.lock.Lock()
	// Close the writing channel so the rest stops
	if pump.Connected {
		close(pump.outgoing)
	}
	// Set user as disconnected
	pump.Connected = false
	pump.lock.Unlock()

	// Something went wrong while writing so disconnect
	pingTicker.Stop()
	pump.ws.Close()
}

// SendMessage schedule message to be send
func (pump *MessagePump) SendMessage(messageType MessageType, pb proto.Message) error {
	// Create buffer to write to with the calculated size required
	messageID := uint64(messageType)
	typebuf := make([]byte, 0, proto.SizeVarint(messageID)+proto.Size(pb))

	response := proto.NewBuffer(typebuf)
	response.EncodeVarint(messageID)
	err := response.Marshal(pb)
	if err != nil {
		return err
	}

	// Send type prefixed message
	pump.lock.RLock()
	if pump.Connected {
		pump.outgoing <- response.Bytes()
	}
	pump.lock.RUnlock()

	return nil
}

// Disconnect schedule disconnection of pump
func (pump *MessagePump) Disconnect() {
	// Close the connection, lock in case we are already disconnecting
	pump.lock.Lock()
	if pump.Connected {
		pump.outgoing <- nil
	}
	pump.lock.Unlock()
	pump.user.OnDisconnect(nil)
}
