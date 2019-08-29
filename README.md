# Introduction
Just a basic chat server I created as a way to learn Go.

Much of the TCP/IP code was taken from https://appliedgo.net/networking/ and subsequently modified.

# Usage
There are two application directories: client and server.
One instance of server should be started as well as one or more instances of client.

# Design
## Server
The server listens on localhost:5001 (which currently isn't configurable) for requests made in my small protocol.

There is a handler function registered for each potential command:
- get rooms
- create room
- join room
- post message
- leave room

A command is parsed by trying to decode data over the wire as an [Action](action.go). It is up to the 
associated handler function to decide if more data is needed.

### Receiving a post
The handler function to receive a post simply decodes data over the wire as a [Message](message.go), determines what room
it is for, and adds it to the room's message channel.

A room has three parts: a name, a message channel, and a slice of users. When a room is created,
all fields are initialized and a goroutine is started that is in an infinite loop trying to consume from the message 
channel. When the goroutine consumes a message (a post was received), it then fans it out to all members in the slice.

When a [User](user.go) is created, a message channel is initialized. A goroutine is created to consume from the channel
and send data back to the client.

By having a channel for both the room and all of it's members, it means that when posting to a given room or sending message
data to a specific user, the main thread is not going to block. Currently the size of both the room and user channel is 10, but that could theoretically be changed based on "load" (load
is in quotes here because this is a toy project).

[Post message diagram](doc/post-diagram.png)

## Client
The client implementation shares a similarity with the server in that there are function handlers
registered for the various responses made by the server. The client is implemented using a finite state machine with 
two states: INIT and IN_ROOM. There are relevant actions the client can perform (via stdin) in each state.  

