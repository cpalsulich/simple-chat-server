package chat_server

type Action struct {
	cmd Command
}

type Command int

const (
	GET_ROOMS   Command = 0
	JOIN_ROOM   Command = 1
	POST        Command = 2
	LEAVE_ROOM  Command = 3
	CREATE_ROOM Command = 4
)