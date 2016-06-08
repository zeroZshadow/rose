package rose

// RoomFront is a wrapper for the room, to ensure syncing is done correctly
type RoomFront struct {
	internal Room
}

// NewRoomFront create a new roomfront for the given room
func NewRoomFront(room Room) *RoomFront {
	return &RoomFront{
		internal: room,
	}
}

// Start start running the internal room
func (front *RoomFront) Start() {
	go roomRun(front.internal)
}

// QueueAddUser Schedule to add a user to the room
func (front *RoomFront) QueueAddUser(user User) {
	msg := userJoining{
		user: user,
	}
	front.internal.internalMessage(msg)
}

// QueueRemoveUser Schedule to remove a user from the room
func (front *RoomFront) QueueRemoveUser(user User) {
	msg := userLeaving{
		user: user,
	}

	front.internal.internalMessage(msg)
}

// PushMessage pushes a messages on the room's messages queueu
func (front *RoomFront) PushMessage(user User, msgType MessageType, message []byte) {
	msg := userMessage{
		user:    user,
		msgType: msgType,
		data:    message,
	}

	front.internal.internalMessage(msg)
}
