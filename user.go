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

		err := gob.NewEncoder(writer).Encode(Action{Name: Post})
		if err != nil {
			log.Print(err)
		}

		err = gob.NewEncoder(writer).Encode(msg)
		if err != nil {
			log.Print(err)
		}

		err = writer.Flush()
		if err != nil {
			log.Print(err)
		}
	}
}
