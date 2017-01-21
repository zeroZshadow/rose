package rose

// RoomFront is a wrapper for the room, to ensure syncing is done correctly
type RoomFront struct {
	ID       RoomID
	internal Room
}

// newRoomFront create a new roomfront for the given room
func newRoomFront(room Room) *RoomFront {
	return &RoomFront{
		internal: room,
		ID:       room.Base().ID,
	}
}

// start start running the internal room
func (front *RoomFront) start() {
	go roomRun(front.internal)
}

// QueueAction will queue a function to be executed in the room.
// This is conncurrency safe because the action will be added to the main channel.
// This channel is read and emptied out in the main room goroutine.
func (front *RoomFront) QueueAction(action Action) {
	front.internal.Base().actionQueue <- action
}

// PushMessage pushes a messages on the room's messages queue.
func (front *RoomFront) PushMessage(user User, msgType MessageType, message []byte) {
	front.QueueAction(func(room Room) {
		room.HandleMessage(user, msgType, message)
	})
}

// addUser Schedule to add a user to the room
func (front *RoomFront) addUser(user User) {
	front.QueueAction(func(room Room) {
		room.AddUser(user)
	})
}

// removeUser Schedule to remove a user from the room
func (front *RoomFront) removeUser(user User) {
	front.QueueAction(func(room Room) {
		room.RemoveUser(user)
	})
}
