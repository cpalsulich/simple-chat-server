package simple_chat_server

import "log"

type Room struct {
	Name    string
	Queue   chan (Message)
	Members []User
}

func NewRoom(name string) *Room {
	r := &Room{
		Name:    name,
		Queue:   make(chan Message, 10),
		Members: make([]User, 0),
	}
	go r.createFanout()

	return r
}

func (r *Room) Post(message *Message) {
	r.Queue <- *message
}

func (r *Room) Join(user *User) {
	r.Members = append(r.Members, *user)
	log.Println("user successfully joined " + user.Id)
	log.Println(r.Members)
}

func (r *Room) Leave(user *User) {
	for i, mem := range r.Members {
		if user.Id == mem.Id {
			// remove element from slice
			r.Members[i] = r.Members[len(r.Members)-1]
			r.Members = r.Members[:len(r.Members)-1]
		}
	}
}

func (r *Room) Close() {
	close(r.Queue)
}

func (r *Room) createFanout() {
	for {
		msg, ok := <-r.Queue
		if ok == false {
			return
		}

		for _, mem := range r.Members {
			log.Printf("adding message %s in room %s for user %s", msg.Message, msg.Room, mem)
			mem.Queue <- msg
		}
	}
}
