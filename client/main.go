package main

import (
	"bufio"
	"encoding/gob"
	chat "github.com/cpalsulich/simple-chat-server"
	"io"
	"log"
	"net"
	"os"
	"strings"
)

func main() {
	conn, err := net.Dial("tcp", "127.0.0.1:5001")

	if err != nil {
		panic(err)
	}
	reader := bufio.NewReader(conn)
	serverHandlers := map[chat.ActionName]ServerHandleFunc{
		chat.GetRooms:  handleGetRooms,
		chat.Post:      handleMessage,
		chat.JoinRoom:  handleJoinRoom,
		chat.LeaveRoom: handleLeaveRoom,
	}

	state := &State{state: initial}

	go handleServerInput(reader, serverHandlers, state)
	rw := bufio.NewReadWriter(bufio.NewReader(os.Stdin), bufio.NewWriter(conn))

	userHandlers := map[ClientState]map[string]UserHandleFunc{
		initial: {
			"create": createRoom,
			"join":   joinRoom,
		},
		inRoom: {
			"post":  post(state),
			"leave": leaveRoom,
		},
	}

	loop(rw, state, userHandlers)
}

type State struct {
	state ClientState
	room  chat.Room
}

type ClientState int

const (
	initial ClientState = 0
	inRoom  ClientState = 1
)

type ServerHandleFunc func(*bufio.Reader, *State)
type UserHandleFunc func(*bufio.Writer, string)

func handleServerInput(reader *bufio.Reader, handlers map[chat.ActionName]ServerHandleFunc, state *State) {
	for {
		action := &chat.Action{}
		if err := gob.NewDecoder(reader).Decode(action); err != nil {
			if err == io.EOF {
				log.Println("Reached EOF - close this connection.\n   ---")
				return
			}
			log.Printf("Error reading command: %v", err)
			return
		}

		handlerFunc, ok := handlers[action.Name]
		if !ok {
			log.Println("Unidentified command")
			continue
		}
		handlerFunc(reader, state)
	}
}

func loop(rw *bufio.ReadWriter, state *State, handlers map[ClientState]map[string]UserHandleFunc) {
	for {
		switch state.state {
		case initial:
			getRooms(rw.Writer)
			log.Println("Join room: 'join <room_name>'")
			log.Println("Create room: 'create <room_name>'")
		case inRoom:
			log.Println("Leave room: 'leave <room_name>'")
			log.Println("Post: 'post <message>'")
		}

		handleUserInput(rw, handlers[state.state])
	}
}

func handleUserInput(rw *bufio.ReadWriter, handlers map[string]UserHandleFunc) {
	input, err := rw.ReadString('\n')
	input = strings.TrimSpace(input)
	if err != nil {
		log.Println("failed to get command")
	}

	cmds := strings.SplitN(input, " ", 2)

	handlerFunc := handlers[cmds[0]]
	if len(cmds) < 2 || handlerFunc == nil {
		log.Println("invalid input: " + input)
		log.Println(cmds[0])
		return
	}

	handlerFunc(rw.Writer, cmds[1])
}

func handleGetRooms(reader *bufio.Reader, _ *State) {
	var rooms []string

	if err := gob.NewDecoder(reader).Decode(&rooms); err != nil {
		log.Print(err)
		return
	}

	if len(rooms) > 0 {
		log.Printf("room list: %s\n", rooms)
	} else {
		log.Println("no rooms created")
	}
}

func handleMessage(reader *bufio.Reader, _ *State) {
	msg := &chat.Message{}
	if err := gob.NewDecoder(reader).Decode(msg); err != nil {
		log.Println("failed to decode message")
	} else {
		log.Printf("%s %s %s", msg.Room, msg.Author, msg.Message)
	}
}

func handleJoinRoom(reader *bufio.Reader, state *State) {
	room := &chat.Room{}
	if err := gob.NewDecoder(reader).Decode(room); err != nil {
		log.Println("failed to decode room")
		return
	}
	log.Println("joined room " + room.Name)
	state.state = inRoom
	state.room = *room
}

func handleLeaveRoom(reader *bufio.Reader, state *State) {
	room := &chat.Room{}
	if err := gob.NewDecoder(reader).Decode(room); err != nil {
		log.Println("failed to decode room")
	} else {
		log.Println("left room " + room.Name)
	}
	state.state = initial
}

func getRooms(w *bufio.Writer) {
	sendRequest(w, chat.GetRooms, nil)
}

func joinRoom(w *bufio.Writer, name string) {
	sendRequest(w, chat.JoinRoom, chat.Room{Name: name})
}

func leaveRoom(w *bufio.Writer, name string) {
	sendRequest(w, chat.LeaveRoom, chat.Room{Name: name})
}

func post(s *State) UserHandleFunc {
	return func(w *bufio.Writer, msg string) {
		sendRequest(w, chat.Post, chat.Message{Message: msg, Room: s.room.Name})
	}
}

func createRoom(w *bufio.Writer, name string) {
	sendRequest(w, chat.CreateRoom, chat.Room{Name: name})
}

func sendRequest(writer *bufio.Writer, actionName chat.ActionName, o interface{}) {
	action := &chat.Action{Name: actionName}
	encoder := gob.NewEncoder(writer)
	if err := encoder.Encode(action); err != nil {
		log.Println("problem writing command")
	}
	if o != nil {
		if err := encoder.Encode(o); err != nil {
			log.Println("problem encoding object for command")
		}
	}
	if err := writer.Flush(); err != nil {
		log.Println("error flushing for command")
	}
}
