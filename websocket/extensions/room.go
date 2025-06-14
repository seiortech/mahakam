package extensions

import (
	"log"
	"sync"

	"github.com/seiortech/mahakam/websocket"
)

type Message struct {
	Data   []byte
	Client *websocket.Client
}

type RoomOption struct {
	RestrictedBroadcast bool
	OnError             func(error)
	OnEnter             func(Message)
	OnLeave             func(Message)
	OnMessage           func(Message)
}

type Room struct {
	Name    string
	clients map[*websocket.Client]bool
	Option  *RoomOption
	Enter   chan Message
	Leave   chan Message
	Message chan Message
	Mutex   sync.RWMutex
}

func NewRoom(name string, option *RoomOption) Room {
	if option == nil {
		option = &RoomOption{
			RestrictedBroadcast: false,
			OnError: func(err error) {
				log.Println(err)
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

func (r *Room) Add(client *websocket.Client) {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()

	r.clients[client] = true
}

func (r *Room) Remove(client *websocket.Client) {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()

	delete(r.clients, client)
}

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

func (r *Room) BroadcastEnter(msg []byte, client *websocket.Client) {
	r.Enter <- Message{
		Data:   msg,
		Client: client,
	}
}

func (r *Room) BroadcastLeave(msg []byte, client *websocket.Client) {
	r.Leave <- Message{
		Data:   msg,
		Client: client,
	}
}

func (r *Room) BroadcastMessage(msg []byte, client *websocket.Client) {
	r.Message <- Message{
		Data:   msg,
		Client: client,
	}
}
