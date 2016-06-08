package rose

import (
	"github.com/golang/protobuf/proto"
)

// UserBase provides message sending
type UserBase struct {
	ID uint64

	// Framework things
	pump *MessagePump
}

// NewUserBase return a new UserBase
func NewUserBase(id uint64, pump *MessagePump) *UserBase {
	return &UserBase{
		ID:   id,
		pump: pump,
	}
}

// SendMessage sends the given message, including type, over the wire
func (base *UserBase) SendMessage(messageType MessageType, pb proto.Message) error {
	return base.pump.SendMessage(messageType, pb)
}

// Base return UserBase
func (base *UserBase) Base() *UserBase {
	return base
}

// Disconnect disconnects the user from the server
func (base *UserBase) Disconnect() {
	base.pump.Disconnect()
}
