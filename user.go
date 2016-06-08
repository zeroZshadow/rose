package rose

import (
	"github.com/golang/protobuf/proto"
)

// User interface for a connection/user
type User interface {
	SendMessage(MessageType, proto.Message) error
	HandlePacket(MessageType, []byte)
	OnDisconnect(error)
	OnConnect()
	Disconnect()
	Base() *UserBase
}
