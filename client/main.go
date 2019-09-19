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
	serverHandlers := map[chat.ActionName]ServerHandleFunc{
		chat.GetRooms:   handleGetRooms,
		chat.Post:       handleMessage,
		chat.JoinRoom:   handleJoinRoom,
		chat.LeaveRoom:  handleLeaveRoom,
		chat.CreateRoom: handleCreateRoom,
	}

	state := &State{state: initial}

	go handleServerInput(bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn)), serverHandlers, state)

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
	loop(bufio.NewReadWriter(bufio.NewReader(os.Stdin), bufio.NewWriter(conn)), state, userHandlers)
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

type ServerHandleFunc func(*bufio.Reader, func(ClientState, chat.Room))
type UserHandleFunc func(*bufio.Writer, string)

func handleServerInput(rw *bufio.ReadWriter, handlers map[chat.ActionName]ServerHandleFunc, state *State) {
	updateStateFunc := updateState(state, rw)
	for {
		action := &chat.Action{}
		if err := gob.NewDecoder(rw).Decode(action); err != nil {
			if err == io.EOF {
				log.Println("Reached EOF - close this connection.\n   ---")
				return
			}
			chat.LogError("Error reading command: %w", err)
			return
		}

		handlerFunc, ok := handlers[action.Name]
		if !ok {
			log.Printf("Unidentified command: %v", action.Name)
			continue
		}

		handlerFunc(rw.Reader, updateStateFunc)
	}
}

func updateState(state *State, rw *bufio.ReadWriter) func(ClientState, chat.Room) {
	return func(cs ClientState, r chat.Room) {
		state.state = cs
		state.room = r

		switch state.state {
		case initial:
			getRooms(rw.Writer)
			log.Println("Join room: 'join <room_name>'")
			log.Println("Create room: 'create <room_name>'")
		case inRoom:
			log.Println("Leave room: 'leave <room_name>'")
			log.Println("Post: 'post <message>'")
		}
	}
}

func loop(rw *bufio.ReadWriter, state *State, handlers map[ClientState]map[string]UserHandleFunc) {
	getRooms(rw.Writer)
	log.Println("Join room: 'join <room_name>'")
	log.Println("Create room: 'create <room_name>'")
	for {
		handleUserInput(rw, state, handlers)
	}
}

func handleUserInput(rw *bufio.ReadWriter, state *State, handlers map[ClientState]map[string]UserHandleFunc) {
	input, err := rw.ReadString('\n')
	input = strings.TrimSpace(input)
	if err != nil {
		chat.LogError("failed to get command: %w", err)
	}

	cmds := strings.SplitN(input, " ", 2)

	handlerFunc := handlers[state.state][cmds[0]]
	if len(cmds) < 2 || handlerFunc == nil {
		log.Println("invalid input: " + input)
		log.Println(cmds[0])
		return
	}

	handlerFunc(rw.Writer, cmds[1])
}

func handleGetRooms(reader *bufio.Reader, _ func(ClientState, chat.Room)) {
	var rooms []string

	if err := gob.NewDecoder(reader).Decode(&rooms); err != nil {
		chat.LogError("failed to decode rooms: %w", err)
		return
	}

	if len(rooms) > 0 {
		log.Printf("room list: [%s]\n", strings.Join(rooms, ", "))
	} else {
		log.Println("no rooms created")
	}
}

func handleMessage(reader *bufio.Reader, _ func(ClientState, chat.Room)) {
	msg := &chat.Message{}
	if err := gob.NewDecoder(reader).Decode(msg); err != nil {
		chat.LogError("failed to decode message: %w", err)
		return
	}
	log.Printf("%s %s %s", msg.Room, msg.Author, msg.Message)
}

func handleJoinRoom(reader *bufio.Reader, updateState func(state ClientState, room chat.Room)) {
	room := &chat.Room{}
	if err := gob.NewDecoder(reader).Decode(room); err != nil {
		chat.LogError("failed to decode room: %w", err)
		return
	}
	log.Println("joined room " + room.Name)
	updateState(inRoom, *room)
}

func handleLeaveRoom(reader *bufio.Reader, updateState func(state ClientState, room chat.Room)) {
	room := &chat.Room{}
	if err := gob.NewDecoder(reader).Decode(room); err != nil {
		chat.LogError("failed to decode room: %w", err)
	} else {
		log.Printf("left room %s", room.Name)
	}
	updateState(initial, chat.Room{})
}

func handleCreateRoom(reader *bufio.Reader, updateState func(state ClientState, room chat.Room)) {
	room := &chat.Room{}
	if err := gob.NewDecoder(reader).Decode(room); err != nil {
		chat.LogError("failed to decode room: %w", err)
	} else {
		log.Printf("room %s created", room.Name)
	}
	updateState(initial, chat.Room{})
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
		chat.LogError("problem writing command: %w", err)
	}
	if o != nil {
		if err := encoder.Encode(o); err != nil {
			chat.LogError("problem encoding object for command: %w", err)
		}
	}
	if err := writer.Flush(); err != nil {
		chat.LogError("error flushing for command: %w", err)
	}
}
