package chat

import (
	"bufio"
	"encoding/gob"
	"errors"
	"io"
	"log"
	"net"
)

type HandleFunc func(*bufio.ReadWriter, *net.Conn)

type ServerEndpoint struct {
	listener net.Listener
	handler  map[ActionName]HandleFunc
}

func NewEndpoint() *ServerEndpoint {
	return &ServerEndpoint{
		handler: map[ActionName]HandleFunc{},
	}
}

func (e *ServerEndpoint) AddHandleFunc(a ActionName, f HandleFunc) {
	e.handler[a] = f
}

func (e *ServerEndpoint) Listen(port string, connectFunc func(conn *net.Conn), disconnectFunc func(conn *net.Conn)) error {
	var err error
	e.listener, err = net.Listen("tcp", "localhost:"+port)
	if err != nil {
		return err
	}
	log.Println("Listen on", e.listener.Addr().String())
	for {
		log.Println("Accept a connection request.")
		conn, err := e.listener.Accept()
		if err != nil {
			LogError("Failed accepting a connection request: %w", err)
			continue
		}
		log.Println("Handle incoming messages.")
		connectFunc(&conn)
		go e.handleMessages(&conn, disconnectFunc)
	}
}

func (e *ServerEndpoint) handleMessages(conn *net.Conn, disconnectFunc func(conn *net.Conn)) {
	rw := bufio.NewReadWriter(bufio.NewReader(*conn), bufio.NewWriter(*conn))
	defer func() {
		disconnectFunc(conn)
		(*conn).Close()
	}()

	for {
		action := &Action{}
		if err := gob.NewDecoder(rw).Decode(action); err != nil {
			if errors.Is(err, io.EOF) {
				return
			}
			LogError("Error reading command: %w", err)
			return
		}
		handleCommand, ok := e.handler[action.Name]
		if !ok {
			log.Println("ActionName is not registered.")
			return
		}
		handleCommand(rw, conn)
	}
}
