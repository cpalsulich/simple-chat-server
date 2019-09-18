package main

import (
	"bufio"
	"encoding/gob"
	scs "github.com/cpalsulich/simple-chat-server"
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
	serverHandlers := make(map[scs.ActionName]ServerHandleFunc)
	serverHandlers[scs.GetRooms] = handleGetRooms
	serverHandlers[scs.Post] = handleMessage
	serverHandlers[scs.JoinRoom] = handleJoinRoom
	serverHandlers[scs.LeaveRoom] = handleLeaveRoom
	state := &State{state: INIT}

	go handleServerInput(reader, serverHandlers, state)
	rw := bufio.NewReadWriter(bufio.NewReader(os.Stdin), bufio.NewWriter(conn))

	userHandlers := map[ClientState]map[string]UserHandleFunc{
		INIT: {
			"create": createRoom,
			"join":   joinRoom,
		},
		IN_ROOM: {
			"post":  post(state),
			"leave": leaveRoom,
		},
	}

	loop(rw, state, userHandlers)
}

type State struct {
	state ClientState
	room  scs.Room
}

type ClientState int

const (
	INIT    ClientState = 0
	IN_ROOM ClientState = 1
)

type ServerHandleFunc func(*bufio.Reader, *State)
type UserHandleFunc func(*bufio.Writer, string)

func handleServerInput(reader *bufio.Reader, handlers map[scs.ActionName]ServerHandleFunc, state *State) {
	for {
		action := &scs.Action{}
		if err := gob.NewDecoder(reader).Decode(action); err != nil {
			if err == io.EOF {
				log.Println("Reached EOF - close this connection.\n   ---")
			} else {
				log.Println("\nError reading command.", err)
			}
			return
		}

		handlerFunc := handlers[action.Name]
		if handlerFunc == nil {
			log.Println("Unidentified command")
			continue
		}
		handlerFunc(reader, state)
	}
}

func loop(rw *bufio.ReadWriter, state *State, handlers map[ClientState]map[string]UserHandleFunc) {
	for {
		switch state.state {
		case INIT:
			getRooms(rw.Writer)
			log.Println("Join room: 'join <room_name>'")
			log.Println("Create room: 'create <room_name>'")
		case IN_ROOM:
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
	rooms := make([]string, 10)

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
	msg := &scs.Message{}
	if err := gob.NewDecoder(reader).Decode(msg); err != nil {
		log.Println("failed to decode message")
	} else {
		log.Printf("%s %s %s", msg.Room, msg.Author, msg.Message)
	}
}

func handleJoinRoom(reader *bufio.Reader, state *State) {
	room := &scs.Room{}
	if err := gob.NewDecoder(reader).Decode(room); err != nil {
		log.Println("failed to decode room")
		return
	}
	log.Println("joined room " + room.Name)
	state.state = IN_ROOM
	state.room = *room
}

func handleLeaveRoom(reader *bufio.Reader, state *State) {
	room := &scs.Room{}
	if err := gob.NewDecoder(reader).Decode(room); err != nil {
		log.Println("failed to decode room")
	} else {
		log.Println("left room " + room.Name)
	}
	state.state = INIT
}

func getRooms(w *bufio.Writer) {
	sendRequest(w, scs.GetRooms, nil)
}

func joinRoom(w *bufio.Writer, name string) {
	sendRequest(w, scs.JoinRoom, scs.Room{Name: name})
}

func leaveRoom(w *bufio.Writer, name string) {
	sendRequest(w, scs.LeaveRoom, scs.Room{Name: name})
}

func post(s *State) UserHandleFunc {
	return func(w *bufio.Writer, msg string) {
		sendRequest(w, scs.Post, scs.Message{Message: msg, Room: s.room.Name})
	}
}

func createRoom(w *bufio.Writer, name string) {
	sendRequest(w, scs.CreateRoom, scs.Room{Name: name})
}

func sendRequest(writer *bufio.Writer, actionName scs.ActionName, o interface{}) {
	action := &scs.Action{Name: actionName}
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
