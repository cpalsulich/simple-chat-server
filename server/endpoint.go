package main

import (
	"bufio"
	"io"
	"log"
	"net"
	"strings"
)

type HandleFunc func(*bufio.ReadWriter, *net.Conn)

type Endpoint struct {
	listener net.Listener
	handler map[string]HandleFunc
}

func NewEndpoint() *Endpoint {
	return &Endpoint{
		handler:  map[string]HandleFunc{},
	}
}

func (e *Endpoint) AddHandleFunc(name string, f HandleFunc) {
	e.handler[name] = f
}

func (e *Endpoint) Listen(port string, connFunc func(conn *net.Conn)) error {
	var err error
	e.listener, err = net.Listen("tcp", "localhost:" + port)
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

func (e *Endpoint) handleMessages(conn *net.Conn) {
	rw := bufio.NewReadWriter(bufio.NewReader(*conn), bufio.NewWriter(*conn))
	defer (*conn).Close()

	for {
		cmd, err := rw.ReadString('\n')
		switch {
		case err == io.EOF:
			log.Println("Reached EOF - close this connection.\n   ---")
			return
		case err != nil:
			log.Println("\nError reading command. Got: '"+cmd+"'\n", err)
			return
		}
		cmd = strings.Trim(cmd, "\n ")
		log.Print("Receive command " + cmd)
		handleCommand, ok := e.handler[cmd]
		if !ok {
			log.Println("Command '" + cmd + "' is not registered.")
			return
		}
		handleCommand(rw, conn)
	}
}