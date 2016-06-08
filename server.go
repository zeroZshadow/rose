package rose

import (
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/hydrogen18/stoppableListener"
	"log"
)

// PacketHandler handle incoming packet
type PacketHandler func(User, []byte)

// UserConstructor create new using with MessagePump
type UserConstructor func(pump *MessagePump) User

// Server Protobuff game server
type Server struct {
	upgrader         websocket.Upgrader
	userConstructors map[string]UserConstructor
	listenWaitGroup  sync.WaitGroup
	listener         *stoppableListener.StoppableListener
	log              log.Logger
}

// New create new server
func New() *Server {
	// Create upgrader
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}

	// Disable origin check
	//DEBUG
	//TODO: change for deployment
	upgrader.CheckOrigin = func(r *http.Request) bool {
		return true
	}

	return &Server{
		upgrader:         upgrader,
		userConstructors: make(map[string]UserConstructor),
	}
}

// Serve Setup the listener and listen
func (serv *Server) Serve(address string) error {
	// Create a Listener
	ls, err := net.Listen("tcp4", address)
	if err != nil {
		return err
	}

	// Create stopable listener
	listener, err := stoppableListener.New(ls)
	if err != nil {
		return err
	}

	// Create http server with router
	httpserver := http.Server{}

	// Create WaitGroup to wait on and be sure the server is done
	serv.listenWaitGroup.Add(1)
	go func() {
		defer serv.listenWaitGroup.Done()
		err := httpserver.Serve(listener)
		if err != nil {
			log.Printf("Error while Serving HTTP: %s\n", err.Error())
		}
	}()

	// Create channel for exit
	stop := make(chan os.Signal)
	signal.Notify(stop, os.Interrupt)

	// Create goroutine to ensure clean exit
	go func() {
		select {
		case <-stop:
			listener.Stop()
		}
	}()

	// Everything went right, save listener
	serv.listener = listener

	return nil
}

// Wait Block until the listener has exited
func (serv *Server) Wait() {
	serv.listenWaitGroup.Wait()
}

// Port Get the port the server is on
func (serv *Server) Port() (uint64, error) {
	address := serv.listener.TCPListener.Addr().String()
	_, port, err := net.SplitHostPort(address)
	if err != nil {
		return 0, err
	}

	return strconv.ParseUint(port, 10, 32)
}

// Listen Setup to handle url with given user
func (serv *Server) Listen(pattern string, constructor UserConstructor) {
	serv.userConstructors[pattern] = constructor
	http.Handle(pattern, serv)
}

// ServerHTTP handles websocket requests from the peer.
func (serv *Server) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		http.Error(writer, "Method not allowed", 405)
		return
	}

	// Upgrade to websocket
	ws, err := serv.upgrader.Upgrade(writer, request, nil)
	if err != nil {
		http.Error(writer, "Could not upgrade to websocket", 400)
		return
	}

	// Get user constructor
	if _, ok := serv.userConstructors[request.RequestURI]; !ok {
		http.Error(writer, "Not found", 404)
		return
	}

	go serv.runPump(ws, serv.userConstructors[request.RequestURI], nil)
}

// Connect connect to target websocket and handle the connection to a user
func (serv *Server) Connect(address string, constructor UserConstructor) (User, error) {
	// Connect the adress, then make a pump with the given user constructor
	ws, _, err := websocket.DefaultDialer.Dial(address, nil)
	if err != nil {
		return nil, err
	}

	// Create return channel
	userchan := make(chan User)

	// Start pump
	go serv.runPump(ws, constructor, userchan)

	// Return the created user to the caller
	return <-userchan, nil
}

func (serv *Server) runPump(ws *websocket.Conn, constructor UserConstructor, userchan chan User) {
	// Setup connection
	pump := NewMessagePump(constructor, ws, serv)

	// Return the newly made user to the caller (though a channel, since we are a goroutine)
	if userchan != nil {
		userchan <- pump.user
	}

	// Start pumps
	go pump.writePump()
	pump.readPump()
}
