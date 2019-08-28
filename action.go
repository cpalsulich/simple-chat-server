package simple_chat_server

type Action struct {
	Name ActionName
}

type ActionName int

const (
	GET_ROOMS   ActionName = 0
	JOIN_ROOM   ActionName = 1
	POST        ActionName = 2
	LEAVE_ROOM  ActionName = 3
	CREATE_ROOM ActionName = 4
)
