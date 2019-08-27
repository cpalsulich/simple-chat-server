package main

import (
	"bufio"
	chat_server "chat-server"
	"encoding/gob"
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
	serverHandlers := make(map[string]ServerHandleFunc)
	serverHandlers["GET_ROOMS"] = handleGetRooms
	serverHandlers["MESSAGE"] = handleMessage
	serverHandlers["JOIN_ROOM"] = handleJoinRoom
	serverHandlers["LEAVE_ROOM"] = handleLeaveRoom
	state := &State{state:INIT}

	go handleServerInput(reader, serverHandlers, state)
	rw := bufio.NewReadWriter(bufio.NewReader(os.Stdin), bufio.NewWriter(conn))

	userHandlers := make(map[ClientState]map[string]UserHandleFunc)
	initHandlers := make(map[string]UserHandleFunc)
	initHandlers["create"] = createRoom
	initHandlers["join"] = joinRoom
	inRoomHandlers := make(map[string]UserHandleFunc)
	inRoomHandlers["post"] = post(state)
	inRoomHandlers["leave"] = leaveRoom
	userHandlers[INIT] = initHandlers
	userHandlers[IN_ROOM] = inRoomHandlers

	loop(rw, state, userHandlers)
}

type State struct {
	state ClientState
	room chat_server.Room
}

type ClientState int

const (
	INIT ClientState = 0
	IN_ROOM ClientState = 1
)

type ServerHandleFunc func(*bufio.Reader, *State)
type UserHandleFunc func(*bufio.Writer, string)

func handleServerInput(reader *bufio.Reader, handlers map[string]ServerHandleFunc, state *State) {
	for {
		cmd, err := reader.ReadString('\n')
		cmd = strings.Trim(cmd, "\n ")
		switch {
		case err == io.EOF:
			log.Println("Reached EOF - close this connection.\n   ---")
			return
		case err != nil:
			log.Println("\nError reading command. Got: '"+cmd+"'\n", err)
			return
		}
		handlerFunc := handlers[cmd]
		if handlerFunc != nil {
			handlerFunc(reader, state)
		} else {
			log.Println("Unidentified command - " + cmd)
		}
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
	} else {
		handlerFunc(rw.Writer, cmds[1])
	}
}

func handleGetRooms(reader *bufio.Reader, state *State) {
	rooms := make([]string, 10)
	err := gob.NewDecoder(reader).Decode(&rooms)
	if err != nil {
		log.Print(err)
		return
	}

	if len(rooms) > 0 {
		log.Printf("room list: %s\n", rooms)
	} else {
		log.Println("no rooms created")
	}
}

func handleMessage(reader *bufio.Reader, state *State) {
	msg := &chat_server.Message{}
	err := gob.NewDecoder(reader).Decode(msg)
	if (err != nil) {
		log.Println("failed to decode message")
	} else {
		log.Printf("%s %s %s", msg.Room, msg.Author, msg.Message)
	}
}

func handleJoinRoom(reader *bufio.Reader, state *State) {
	room := &chat_server.Room{}
	err := gob.NewDecoder(reader).Decode(room)
	if (err != nil) {
		log.Println("failed to decode room")
	} else {
		log.Println("joined room " + room.Name)
	}
	state.state = IN_ROOM
	state.room = *room
}

func handleLeaveRoom(reader *bufio.Reader, state *State) {
	room := &chat_server.Room{}
	err := gob.NewDecoder(reader).Decode(room)
	if (err != nil) {
		log.Println("failed to decode room")
	} else {
		log.Println("left room " + room.Name)
	}
	state.state = INIT
}

func getRooms(w *bufio.Writer) {
	sendRequest(w, "GET_ROOMS", nil)
}

func joinRoom(w *bufio.Writer, name string) {
	sendRequest(w, "JOIN_ROOM", chat_server.Room{Name:name})
}

func leaveRoom(w *bufio.Writer, name string) {
	sendRequest(w, "LEAVE_ROOM", chat_server.Room{Name:name})
}

func post(s *State) UserHandleFunc {
	return func (w *bufio.Writer, msg string) {
		sendRequest(w, "POST", chat_server.Message{Message:msg, Room:s.room.Name})
	}
}

func createRoom(w *bufio.Writer, name string) {
	sendRequest(w, "CREATE_ROOM", chat_server.Room{Name:name})
}

func sendRequest(writer *bufio.Writer, cmd string, o interface{}) {
	_, err := writer.WriteString(cmd + "\n")
	if err != nil {
		log.Println("problem writing command " + cmd)
	}
	if o != nil {
		err = gob.NewEncoder(writer).Encode(o)
		if err != nil {
			log.Println("problem encoding object for command " + cmd)
		}
	}
	err = writer.Flush()
	if err != nil {
		log.Println("error flushing for command " + cmd)
	}
}
