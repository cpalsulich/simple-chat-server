package main

import (
	"bufio"
	"chat-server"
	"encoding/gob"
	"log"
	"net"
)

func main() {
	clients := make(map[string]*chat_server.User)
	rooms := make(map[string]*chat_server.Room)
	err := server(clients, rooms)

	if err != nil {
		panic(err)
	}
}

func server(users map[string]*chat_server.User, rooms map[string]*chat_server.Room) error {
	endpoint := NewEndpoint()
	endpoint.AddHandleFunc("GET_ROOMS", handleGetRooms(rooms))
	endpoint.AddHandleFunc("JOIN_ROOM", handleJoinRoom(rooms, users))
	endpoint.AddHandleFunc("POST", handlePost(rooms))
	endpoint.AddHandleFunc("LEAVE_ROOM", handleLeaveRoom(rooms, users))
	endpoint.AddHandleFunc("CREATE_ROOM", handleCreateRoom(rooms))
	return endpoint.Listen("5001", handleConnect(users))
}

func handleConnect(users map[string]*chat_server.User) func(conn *net.Conn) {
	return func(conn *net.Conn) {
		user := chat_server.NewUser(conn)
		log.Printf("user connected %s", user)
		users[user.Id] = user
	}
}

func handleGetRooms(rooms map[string]*chat_server.Room) func(*bufio.ReadWriter, *net.Conn) {
	return func (rw *bufio.ReadWriter, conn *net.Conn) {
		keys := make([]string, len(rooms))
		i := 0
		for k := range rooms {
			keys[i] = k
			i++
		}

		log.Println("sending get rooms response")

		_, err := rw.WriteString("GET_ROOMS\n")
		if err != nil {
			log.Print(err)
		}

		err = gob.NewEncoder(rw).Encode(keys)
		if err != nil {
			log.Print(err)
		}

		err = rw.Flush()
		if err != nil {
			log.Print(err)
		}
	}
}

func handleJoinRoom(rooms map[string]*chat_server.Room, users map[string]*chat_server.User) func(*bufio.ReadWriter, *net.Conn) {
	return func (rw *bufio.ReadWriter, conn *net.Conn) {
		r := &chat_server.Room{}
		err := gob.NewDecoder(rw).Decode(r)
		if err != nil {
			log.Print(err)
		}

		room := rooms[r.Name]
		if room == nil {
			log.Println("joining non-existent room " + r.Name)
			return
		}
		u := chat_server.NewUser(conn)

		log.Printf("user %s joining room %s", u, r)

		room.Join(users[u.Id])

		_, err = rw.WriteString("JOIN_ROOM\n")
		if err != nil {
			log.Print(err)
		}

		err = gob.NewEncoder(rw).Encode(r)
		if err != nil {
			log.Printf("failed to encode room: %s", err)
		}
		err = rw.Flush()
		if err != nil {
			log.Print(err)
		}
	}
}

func handlePost(rooms map[string]*chat_server.Room) func(*bufio.ReadWriter, *net.Conn) {
	return func(rw *bufio.ReadWriter, conn *net.Conn) {
		msg := &chat_server.Message{}
		err := gob.NewDecoder(rw).Decode(msg)
		if err != nil {
			log.Print(err)
		}

		r := rooms[msg.Room]
		msg.Author = chat_server.NewUser(conn).Id
		if r != nil {
			log.Printf("post message %s to room %s", msg.Message, msg.Room)
			r.Post(msg)
		} else {
			log.Printf("couldn't find room (%s) for post\n", msg.Room)
		}

	}
}

func handleLeaveRoom(rooms map[string]*chat_server.Room, users map[string]*chat_server.User) func(*bufio.ReadWriter, *net.Conn) {
	return func(rw *bufio.ReadWriter, conn *net.Conn) {
		r := &chat_server.Room{}
		err := gob.NewDecoder(rw).Decode(r)
		if err != nil {
			log.Print(err)
		}

		room := rooms[r.Name]
		c := chat_server.NewUser(conn)

		room.Leave(users[c.Id])

		_, err = rw.WriteString("LEAVE_ROOM\n")
		if err != nil {
			log.Print(err)
		}

		err = gob.NewEncoder(rw).Encode(room)
		if err != nil {
			log.Print(err)
		}
		rw.Flush()
		if err != nil {
			log.Print(err)
		}
	}
}

func handleCreateRoom(rooms map[string]*chat_server.Room) func(*bufio.ReadWriter, *net.Conn) {
	return func(rw *bufio.ReadWriter, conn *net.Conn) {
		log.Println("new room")
		r := &chat_server.Room{}
		err := gob.NewDecoder(rw).Decode(r)
		if err != nil {
			log.Print(err)
		}

		log.Println("room name " + r.Name)
		rooms[r.Name] = chat_server.NewRoom(r.Name)
	}
}
