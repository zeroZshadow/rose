package rose

import (
	"time"

	"github.com/golang/protobuf/proto"
)

// TODO Maby var instead? so it's changeable
const (
	maxRoomMessages = 20
	//updateRate      = time.Second / 60
)

// RoomBase will have to be concurrent
// all functions altering the room or dealing with the room will have to be private.
// With the exception to the functions passing the events along
type RoomBase struct {
	ID    RoomID
	Users map[User]struct{}
	Owner User

	internalQueue chan roomMessage
	tickrate      time.Duration
	ticker        *time.Ticker
	destroying    bool
	exit          chan bool
}

// NewRoomBase returns a new RoomBase
func NewRoomBase(id RoomID, tickrate time.Duration) (room *RoomBase) {
	// Create the new room
	room = &RoomBase{
		ID:    id,
		Users: make(map[User]struct{}),

		internalQueue: make(chan roomMessage, maxRoomMessages),
		tickrate:      tickrate,
		ticker:        time.NewTicker(tickrate),
		exit:          make(chan bool),
	}

	return
}

// AddUser adds a user to the room
func (room *RoomBase) AddUser(user User) {
	// Add new user
	room.Users[user] = struct{}{}

	// If the room is not claimed yet, claim it
	if room.Owner == nil {
		room.Owner = user
	}
}

// RemoveUser removes a user from the room
func (room *RoomBase) RemoveUser(user User) {
	// Remove user
	delete(room.Users, user)

	// If the room is claimed by this user, transfer the claim
	// If no one can pickup the claim, destroy the room
	if room.Owner == user {
		if len(room.Users) > 0 {
			// XXX: Hack to get first key from map
			for k := range room.Users {
				room.Owner = k
				break
			}
		} else {
			// Remove owner, mark from destruction
			room.Owner = nil
			room.destroying = true
		}
	}
}

// Cleanup Actually cleans the room, SHOULD ONLY BE RUN INSIDE RoomRun !
func (room *RoomBase) Cleanup() {
	// Cleanup gracefully
	room.ticker.Stop()
}

// Base base class for room
func (room *RoomBase) Base() *RoomBase {
	return room
}

// internalMessage push internal room message on queue
func (room *RoomBase) internalMessage(message roomMessage) {
	room.internalQueue <- message
}

// Broadcast send a message to all users in the room
func (room *RoomBase) Broadcast(msgType MessageType, message proto.Message) {
	for user := range room.Users {
		user.SendMessage(msgType, message)
	}
}

// BroadcastExcluded send a message to all users in the room, except given user
func (room *RoomBase) BroadcastExcluded(msgType MessageType, message proto.Message, excludedUser User) {
	for user := range room.Users {
		if user != excludedUser {
			user.SendMessage(msgType, message)
		}
	}
}
