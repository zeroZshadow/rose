package rose

import "sync"
import "errors"
import "fmt"

type roomConstructor func(RoomID) Room

// Lobby collection of rooms
type Lobby struct {
	server *Server
	rooms  map[RoomID]roomInfo
	sync.RWMutex
}

type roomInfo struct {
	room      *RoomFront
	userCount int
}

// RoomLobby global lobby instance
var RoomLobby *Lobby

func initializeLobby(server *Server) {
	RoomLobby = &Lobby{
		server: server,
		rooms:  make(map[RoomID]roomInfo),
	}
}

// NewRoom Create a new room and save it in the lobby
func (lobby *Lobby) NewRoom(id RoomID, constructor roomConstructor) *RoomFront {
	// Lock room list for writing
	lobby.Lock()
	defer lobby.Unlock()

	// Only create a new room if the ID is not yet taken
	var front *RoomFront
	if _, ok := lobby.rooms[id]; !ok {
		// Create the new room
		front = newRoomFront(constructor(id))
		lobby.rooms[id] = roomInfo{
			room: front,
		}
	} else {
		lobby.server.log.Println("Trying to create already existing room ", id)
	}

	// Start up the goroutine that will handle the room
	front.start()

	return front
}

// JoinRoom Add user to room
func (lobby *Lobby) JoinRoom(id RoomID, user User) error {
	lobby.Lock()
	defer lobby.Unlock()

	roomInfo, ok := lobby.rooms[id]
	if !ok {
		return errors.New(fmt.Sprint("Lobby is unable to join room", id))
	}

	roomInfo.userCount++
	roomInfo.room.addUser(user)

	return nil
}

// LeaveRoom Remove user from room and cleanup if required
func (lobby *Lobby) LeaveRoom(id RoomID, user User) error {
	lobby.Lock()
	defer lobby.Unlock()

	roomInfo, ok := lobby.rooms[id]
	if !ok {
		return errors.New(fmt.Sprint("Lobby is unable to leave room", id))
	}

	roomInfo.userCount--
	roomInfo.room.removeUser(user)

	// Destroy room when empty
	if roomInfo.userCount == 0 {
		delete(lobby.rooms, id)
		roomInfo.room.QueueAction(func(room Room) {
			room.Base().destroying = true
		})
	}

	return nil
}
