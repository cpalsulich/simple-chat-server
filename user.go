package chat

import (
	"bufio"
	"encoding/gob"
	"log"
	"net"
)

type User struct {
	ID    string
	Queue chan (Message)
	Conn  *net.Conn
}

func NewUser(conn *net.Conn) *User {

	user := &User{
		ID:    (*conn).RemoteAddr().String(),
		Queue: make(chan Message, 10),
		Conn:  conn,
	}
	go user.createClientConsumer()
	return user
}

func (u *User) createClientConsumer() {
	writer := bufio.NewWriter(*u.Conn)
	for {
		msg, ok := <-u.Queue
		if ok == false {
			return
		}

		log.Printf("receiving message %s in user %s queue", msg.Message, u.ID)

		if err := gob.NewEncoder(writer).Encode(Action{Name: Post}); err != nil {
			LogError("error encoding action: %w", err)
			return
		}

		if err := gob.NewEncoder(writer).Encode(msg); err != nil {
			LogError("error encoding message: %w", err)
			return
		}

		if err := writer.Flush(); err != nil {
			LogError("error flushing: %w", err)
		}
	}
}
