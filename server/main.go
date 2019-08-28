package main

import (
	"bufio"
	"encoding/gob"
	"log"
	"net"
	scs "simple-chat-server"
)

func main() {
	clients := make(map[string]*scs.User)
	rooms := make(map[string]*scs.Room)
	err := server(clients, rooms)

	if err != nil {
		panic(err)
	}
}

func server(users map[string]*scs.User, rooms map[string]*scs.Room) error {
	endpoint := scs.NewEndpoint()
	endpoint.AddHandleFunc(scs.GET_ROOMS, handleGetRooms(rooms))
	endpoint.AddHandleFunc(scs.JOIN_ROOM, handleJoinRoom(rooms, users))
	endpoint.AddHandleFunc(scs.POST, handlePost(rooms))
	endpoint.AddHandleFunc(scs.LEAVE_ROOM, handleLeaveRoom(rooms, users))
	endpoint.AddHandleFunc(scs.CREATE_ROOM, handleCreateRoom(rooms))
	return endpoint.Listen("5001", handleConnect(users))
}

func handleConnect(users map[string]*scs.User) func(conn *net.Conn) {
	return func(conn *net.Conn) {
		user := scs.NewUser(conn)
		log.Printf("user connected %s", user)
		users[user.Id] = user
	}
}

func handleGetRooms(rooms map[string]*scs.Room) func(*bufio.ReadWriter, *net.Conn) {
	return func(rw *bufio.ReadWriter, conn *net.Conn) {
		keys := make([]string, len(rooms))
		i := 0
		for k := range rooms {
			keys[i] = k
			i++
		}

		log.Println("sending get rooms response")
		enc := gob.NewEncoder(rw)

		err := enc.Encode(scs.Action{Name: scs.GET_ROOMS})
		if err != nil {
			log.Print(err)
		}

		err = enc.Encode(keys)
		if err != nil {
			log.Print(err)
		}

		err = rw.Flush()
		if err != nil {
			log.Print(err)
		}
	}
}

func handleJoinRoom(rooms map[string]*scs.Room, users map[string]*scs.User) func(*bufio.ReadWriter, *net.Conn) {
	return func(rw *bufio.ReadWriter, conn *net.Conn) {
		r := &scs.Room{}
		err := gob.NewDecoder(rw).Decode(r)
		if err != nil {
			log.Print(err)
		}

		room := rooms[r.Name]
		if room == nil {
			log.Println("joining non-existent room " + r.Name)
			return
		}
		u := scs.NewUser(conn)

		log.Printf("user %s joining room %s", u, r)

		room.Join(users[u.Id])

		enc := gob.NewEncoder(rw)

		err = enc.Encode(scs.Action{Name: scs.JOIN_ROOM})
		if err != nil {
			log.Print(err)
		}

		err = enc.Encode(r)
		if err != nil {
			log.Printf("failed to encode room: %s", err)
		}
		err = rw.Flush()
		if err != nil {
			log.Print(err)
		}
	}
}

func handlePost(rooms map[string]*scs.Room) func(*bufio.ReadWriter, *net.Conn) {
	return func(rw *bufio.ReadWriter, conn *net.Conn) {
		msg := &scs.Message{}
		err := gob.NewDecoder(rw).Decode(msg)
		if err != nil {
			log.Print(err)
		}

		r := rooms[msg.Room]
		msg.Author = scs.NewUser(conn).Id
		if r != nil {
			log.Printf("post message %s to room %s", msg.Message, msg.Room)
			r.Post(msg)
		} else {
			log.Printf("couldn't find room (%s) for post\n", msg.Room)
		}

	}
}

func handleLeaveRoom(rooms map[string]*scs.Room, users map[string]*scs.User) func(*bufio.ReadWriter, *net.Conn) {
	return func(rw *bufio.ReadWriter, conn *net.Conn) {
		r := &scs.Room{}
		err := gob.NewDecoder(rw).Decode(r)
		if err != nil {
			log.Print(err)
		}

		room := rooms[r.Name]
		c := scs.NewUser(conn)

		room.Leave(users[c.Id])

		enc := gob.NewEncoder(rw)

		err = enc.Encode(scs.Action{Name: scs.LEAVE_ROOM})
		if err != nil {
			log.Print(err)
		}

		err = enc.Encode(room)
		if err != nil {
			log.Print(err)
		}
		err = rw.Flush()
		if err != nil {
			log.Print(err)
		}
	}
}

func handleCreateRoom(rooms map[string]*scs.Room) func(*bufio.ReadWriter, *net.Conn) {
	return func(rw *bufio.ReadWriter, conn *net.Conn) {
		log.Println("new room")
		r := &scs.Room{}
		err := gob.NewDecoder(rw).Decode(r)
		if err != nil {
			log.Print(err)
		}

		log.Println("room name " + r.Name)
		rooms[r.Name] = scs.NewRoom(r.Name)
	}
}
