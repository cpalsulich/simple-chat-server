package main

import (
	"bufio"
	"encoding/gob"
	scs "github.com/cpalsulich/simple-chat-server"
	"log"
	"net"
)

func main() {
	clients := make(map[string]*scs.User)
	rooms := make(map[string]*scs.Room)

	if err := startServer(clients, rooms); err != nil {
		panic(err)
	}
}

type server struct {
	rooms      map[string]*scs.Room
	users      map[string]*scs.User
	creating   chan scs.Room
	connecting chan scs.User
}

func startServer(users map[string]*scs.User, rooms map[string]*scs.Room) error {
	server := server{
		rooms:      rooms,
		users:      users,
		creating:   make(chan scs.Room, 10),
		connecting: make(chan scs.User, 10),
	}
	go server.consumeCreating()
	go server.consumeConnecting()

	endpoint := scs.NewEndpoint()
	endpoint.AddHandleFunc(scs.GetRooms, server.handleGetRooms)
	endpoint.AddHandleFunc(scs.JoinRoom, server.handleJoinRoom)
	endpoint.AddHandleFunc(scs.Post, server.handlePost)
	endpoint.AddHandleFunc(scs.LeaveRoom, server.handleLeaveRoom)
	endpoint.AddHandleFunc(scs.CreateRoom, server.handleCreateRoom)
	return endpoint.Listen("5001", server.handleConnect)
}

func (s *server) consumeCreating() {
	for {
		r, ok := <-s.creating
		if ok == false {
			return
		}
		log.Println("new room")
		log.Println("room name " + r.Name)
		s.rooms[r.Name] = scs.NewRoom(r.Name)
	}
}

func (s *server) consumeConnecting() {
	for {
		u, ok := <-s.connecting
		if ok == false {
			return
		}
		log.Printf("user connected %s", u)
		s.users[u.ID] = &u
	}
}

func (s *server) handleConnect(conn *net.Conn) {
	user := scs.NewUser(conn)
	s.connecting <- *user
}

func (s *server) handleGetRooms(rw *bufio.ReadWriter, conn *net.Conn) {
	keys := make([]string, len(s.rooms))
	i := 0
	for k := range s.rooms {
		keys[i] = k
		i++
	}

	log.Println("sending get rooms response")
	enc := gob.NewEncoder(rw)

	if err := enc.Encode(scs.Action{Name: scs.GetRooms}); err != nil {
		log.Print(err)
	}

	if err := enc.Encode(keys); err != nil {
		log.Print(err)
	}

	if err := rw.Flush(); err != nil {
		log.Print(err)
	}
}

func (s *server) handleJoinRoom(rw *bufio.ReadWriter, conn *net.Conn) {
	r := &scs.Room{}

	if err := gob.NewDecoder(rw).Decode(r); err != nil {
		log.Print(err)
	}

	room := s.rooms[r.Name]
	if room == nil {
		log.Println("joining non-existent room " + r.Name)
		return
	}
	u := scs.NewUser(conn)
	log.Printf("user %s joining room %s", u, r)
	room.Join(s.users[u.ID])
	enc := gob.NewEncoder(rw)

	if err := enc.Encode(scs.Action{Name: scs.JoinRoom}); err != nil {
		log.Print(err)
	}

	if err := enc.Encode(r); err != nil {
		log.Printf("failed to encode room: %s", err)
	}

	if err := rw.Flush(); err != nil {
		log.Print(err)
	}
}

func (s *server) handlePost(rw *bufio.ReadWriter, conn *net.Conn) {
	msg := &scs.Message{}

	if err := gob.NewDecoder(rw).Decode(msg); err != nil {
		log.Print(err)
	}

	r := s.rooms[msg.Room]
	if r == nil {
		log.Printf("couldn't find room (%s) for post\n", msg.Room)
		return
	}
	msg.Author = scs.NewUser(conn).ID
	r.Post(msg)
	log.Printf("post message %s to room %s", msg.Message, msg.Room)
}

func (s *server) handleLeaveRoom(rw *bufio.ReadWriter, conn *net.Conn) {
	r := &scs.Room{}
	if err := gob.NewDecoder(rw).Decode(r); err != nil {
		log.Print(err)
		return
	}

	room := s.rooms[r.Name]
	c := scs.NewUser(conn)
	room.Leave(s.users[c.ID])
	enc := gob.NewEncoder(rw)

	if err := enc.Encode(scs.Action{Name: scs.LeaveRoom}); err != nil {
		log.Print(err)
		return
	}

	if err := enc.Encode(room); err != nil {
		log.Print(err)
		return
	}

	if err := rw.Flush(); err != nil {
		log.Print(err)
	}
}

func (s *server) handleCreateRoom(rw *bufio.ReadWriter, conn *net.Conn) {
	r := &scs.Room{}
	if err := gob.NewDecoder(rw).Decode(r); err != nil {
		log.Print(err)
		return
	}
	s.creating <- *r
}
