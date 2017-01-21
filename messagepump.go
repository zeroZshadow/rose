package rose

import (
	"net"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/gorilla/websocket"
)

const (
	// writeWait amount of time that can pass after a packet write times out.
	writeTimeout = 10 * time.Second
	// pongWait amount of time that can pass after a packet read times out (used for ping).
	readTimeout = 10 * time.Second
	// pingPeriod time between ping messages.
	pingPeriod = (readTimeout * 9) / 10
	// maxMessageSize maximum packet size in bytes.
	maxMessageSize = 4096
	// maxMessages maximum messages that can be queued for sending before blocking.
	maxMessages = 8
)

// MessageType message id type
type MessageType uint64

// MessagePump pump used for reading and writing on a websocket.
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
	if err := pump.ws.SetReadDeadline(time.Now().Add(readTimeout)); err != nil {
		pump.server.log.Println(err)
		return
	}

	pump.ws.SetPongHandler(func(string) error {
		return pump.ws.SetReadDeadline(time.Now().Add(readTimeout))
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
	err := pump.ws.SetWriteDeadline(time.Now().Add(writeTimeout))
	if err != nil {
		return err
	}

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

	// Close connection
	pingTicker.Stop()

	if err := pump.ws.Close(); err != nil {
		pump.server.log.Println(err)
	}
}

// SendMessage schedule message to be send
func (pump *MessagePump) SendMessage(messageType MessageType, pb proto.Message) error {
	// Create buffer to write to with the calculated size required
	messageID := uint64(messageType)

	// TODO Stop constantly making buffers
	typebuf := make([]byte, 0, proto.SizeVarint(messageID)+proto.Size(pb))
	response := proto.NewBuffer(typebuf)

	// Serialize
	if err := response.EncodeVarint(messageID); err != nil {
		return err
	}
	if err := response.Marshal(pb); err != nil {
		return err
	}

	// Send type prefixed message
writeLoop:
	for {
		pump.lock.Lock()
		if !pump.Connected {
			break writeLoop
		}

		select {
		case pump.outgoing <- response.Bytes():
			break writeLoop
		default:
		}
		pump.lock.Unlock()
	}
	pump.lock.Unlock()

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
}
