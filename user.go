package rose

import (
	"github.com/golang/protobuf/proto"
)

// UserID a user identifier
type UserID uint64

// User interface for a connection/user
type User interface {
	SendMessage(MessageType, proto.Message) error
	HandlePacket(MessageType, []byte)
	OnDisconnect(error)
	OnConnect()
	Disconnect()
	Base() *UserBase
}
