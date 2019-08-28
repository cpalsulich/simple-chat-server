package simple_chat_server

import (
	"bufio"
	"encoding/gob"
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

func (e *ServerEndpoint) Listen(port string, connFunc func(conn *net.Conn)) error {
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
			log.Println("Failed accepting a connection request:", err)
			continue
		}
		log.Println("Handle incoming messages.")
		connFunc(&conn)
		go e.handleMessages(&conn)
	}
}

func (e *ServerEndpoint) handleMessages(conn *net.Conn) {
	rw := bufio.NewReadWriter(bufio.NewReader(*conn), bufio.NewWriter(*conn))
	defer (*conn).Close()

	for {
		action := &Action{}
		err := gob.NewDecoder(rw).Decode(action)
		switch {
		case err == io.EOF:
			log.Println("Reached EOF - close this connection.\n   ---")
			return
		case err != nil:
			log.Println("\nError reading command.", err)
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
