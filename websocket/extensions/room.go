package extensions

import (
	"log"
	"sync"

	"github.com/seiortech/mahakam/websocket"
)

// Message represents a message sent in the room.
type Message struct {
	Data   []byte
	Client *websocket.Client
}

// RoomOption contains options for configuring the behavior of a Room.
type RoomOption struct {
	RestrictedBroadcast bool
	OnError             func(error)
	OnEnter             func(Message)
	OnLeave             func(Message)
	OnMessage           func(Message)
}

// Room represents a chat room that manages clients and broadcasts messages.
type Room struct {
	Name    string
	clients map[*websocket.Client]bool
	Option  *RoomOption
	Enter   chan Message
	Leave   chan Message
	Message chan Message
	Mutex   sync.RWMutex
}

// NewRoom creates a new Room with the specified name and options.
func NewRoom(name string, option *RoomOption) Room {
	if option == nil {
		option = &RoomOption{
			RestrictedBroadcast: false,
			OnError: func(err error) {
				log.Println(err)
			},
			OnEnter: func(msg Message) {
			},
			OnLeave: func(msg Message) {
			},
			OnMessage: func(msg Message) {
			},
		}
	}

	return Room{
		Name:    name,
		clients: make(map[*websocket.Client]bool),
		Option:  option,
		Enter:   make(chan Message),
		Leave:   make(chan Message),
		Message: make(chan Message),
		Mutex:   sync.RWMutex{},
	}
}

// Run starts the room's event loop, processing enter, leave, and message events.
func (r *Room) Run() {
	for {
		select {
		case msg := <-r.Enter:
			r.Add(msg.Client)
			r.Broadcast(msg.Data, websocket.TEXT)
			r.Option.OnEnter(msg)
		case msg := <-r.Leave:
			r.Remove(msg.Client)
			r.Broadcast(msg.Data, websocket.TEXT)
			r.Option.OnLeave(msg)
		case msg := <-r.Message:
			r.Broadcast(msg.Data, websocket.TEXT)
			r.Option.OnMessage(msg)
		}
	}
}

// Add adds a client to the room and broadcasts the enter message.
func (r *Room) Add(client *websocket.Client) {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()

	r.clients[client] = true
}

// Remove removes a client from the room and broadcasts the leave message.
func (r *Room) Remove(client *websocket.Client) {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()

	delete(r.clients, client)
}

// Close closes the room, disconnecting all clients, closing channels, and sending a leave message with normal closure code.
func (r *Room) Close() {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()

	for client := range r.clients {
		err := client.Close(nil, websocket.STATUS_CLOSE_NORMAL_CLOSURE)
		if err != nil {
			r.Option.OnError(err)
		}
	}

	r.clients = make(map[*websocket.Client]bool)

	close(r.Enter)
	close(r.Leave)
	close(r.Message)
}

// Broadcast sends a message to all clients in the room.
func (r *Room) Broadcast(message []byte, opcode websocket.Opcode) error {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()

	for client := range r.clients {
		_, err := client.Write(message, opcode)
		if err != nil {
			if r.Option.RestrictedBroadcast {
				return err
			}

			r.Option.OnError(err)
		}
	}

	return nil
}

// BroadcastEnter broadcast a message and add the client to the room.
func (r *Room) BroadcastEnter(msg []byte, client *websocket.Client) {
	r.Enter <- Message{
		Data:   msg,
		Client: client,
	}
}

// BroadcastLeave broadcast a message and remove the client from the room.
func (r *Room) BroadcastLeave(msg []byte, client *websocket.Client) {
	r.Leave <- Message{
		Data:   msg,
		Client: client,
	}
}

// BroadcastMessage broadcasts a message to all clients in the room.
func (r *Room) BroadcastMessage(msg []byte, client *websocket.Client) {
	r.Message <- Message{
		Data:   msg,
		Client: client,
	}
}

// Count returns the number of clients currently in the room.
func (r *Room) Count() int {
	r.Mutex.RLock()
	defer r.Mutex.RUnlock()

	return len(r.clients)
}

// Has checks if a client is in the room.
func (r *Room) Has(client *websocket.Client) bool {
	r.Mutex.RLock()
	defer r.Mutex.RUnlock()

	_, exists := r.clients[client]
	return exists
}

// Clients returns a slice of all clients currently in the room.
func (r *Room) Clients() []*websocket.Client {
	r.Mutex.RLock()
	defer r.Mutex.RUnlock()

	clients := make([]*websocket.Client, 0, len(r.clients))
	for client := range r.clients {
		clients = append(clients, client)
	}
	return clients
}
