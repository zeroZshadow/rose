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
	internalMessage(roomMessage)
	Broadcast(MessageType, proto.Message)
	BroadcastExcluded(MessageType, proto.Message, User)
	Cleanup()
	Base() *RoomBase
}

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
		// Check for exit first
		select {
		case <-base.exit:
			// Die
			break
		default:
			// Handle messages or tick
			select {
			case <-base.ticker.C:
				room.Tick()
				runtime.Gosched()
			case message := <-base.internalQueue:
				handleMessage(room, message)
			}
		}
	}
}

func handleMessage(room Room, message roomMessage) {
	switch m := message.(type) {
	case userMessage:
		room.HandleMessage(m.user, m.msgType, m.data)
	case userJoining:
		room.AddUser(m.user)
	case userLeaving:
		room.RemoveUser(m.user)
	}
}
