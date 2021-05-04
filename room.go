package chat

import "log"

type Room struct {
	Name     string
	Messages chan Message
	Members  map[string]User
	joining  chan User
	leaving  chan User
}

func NewRoom(name string) *Room {
	r := &Room{
		Name:     name,
		Messages: make(chan Message, 10),
		Members:  map[string]User{},
		joining:  make(chan User, 10),
		leaving:  make(chan User, 10),
	}
	go r.consumeMessages()
	go r.consumeJoiners()
	go r.consumeLeavers()

	return r
}

func (r *Room) Post(message *Message) {
	r.Messages <- *message
}

func (r *Room) Join(user *User) {
	r.joining <- *user
}

func (r *Room) Leave(user *User) {
	r.leaving <- *user
}

func (r *Room) Close() {
	close(r.Messages)
}

func (r *Room) consumeMessages() {
	for {
		msg, ok := <-r.Messages
		if ok == false {
			return
		}

		log.Printf("room %s members size %d", r.Name, len(r.Members))
		for _, mem := range r.Members {
			log.Printf("adding message %s in room %s for user %s", msg.Message, msg.Room, mem)
			mem.Queue <- msg
		}
	}
}

func (r *Room) consumeJoiners() {
	for {
		user, ok := <-r.joining
		if ok == false {
			return
		}
		r.Members[user.ID] = user
		log.Println("user joined " + user.ID)
		log.Println(r.Members)
	}
}

func (r *Room) consumeLeavers() {
	for {
		user, ok := <-r.leaving
		if ok == false {
			return
		}
		delete(r.Members, user.ID)
		log.Println("user left " + user.ID)
	}
}
