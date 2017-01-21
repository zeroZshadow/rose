package rose

import (
	"log"
	"runtime"

	"github.com/golang/protobuf/proto"
)

// RoomID unique room id type
type RoomID uint64

// Room interface describing a game room.
type Room interface {
	// Overwritable
	Tick()
	AddUser(User)
	RemoveUser(User)
	HandleMessage(User, MessageType, []byte)
	Broadcast(MessageType, proto.Message)
	BroadcastExcluded(MessageType, proto.Message, User)
	Cleanup()
	Base() *RoomBase
}

// Action to run on the room's goroutine
// This can be used to interact with the rooms internals
type Action func(Room)

// roomRun a goroutine. Runs the room code until destruction of the room
func roomRun(room Room) {
	// Make sure we clean up
	defer room.Cleanup()

	// Set up crash recovery
	defer func() {
		if r := recover(); r != nil {
			//TODO Move to handler that is setable from the outside!
			trace := make([]byte, 1024)
			runtime.Stack(trace, false)
			log.Printf("Recovered from a panic\n%+v\n%s\n", r, trace)
			return
		}
	}()

	// Get RoomBase
	base := room.Base()

	// Start processing room
	for !base.destroying {
		select {
		case <-base.ticker.C:
			room.Tick()
			runtime.Gosched()
		case action := <-base.actionQueue:
			action(room)
		}
	}
}
