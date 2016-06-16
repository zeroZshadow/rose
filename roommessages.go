package rose

// roomMessage Interface for internal room messages.
type roomMessage interface {
	roomMessage()
}

// userJoining user joins the room.
type userJoining struct {
	user User
}

func (userJoining) roomMessage() {}

// userLeaving user leaves the room.
type userLeaving struct {
	user User
}

func (userLeaving) roomMessage() {}

// userMessage messages from user that needs to be parsed by the room.
type userMessage struct {
	user    User
	msgType MessageType
	data    []byte
}

func (userMessage) roomMessage() {}
