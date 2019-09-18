package chat

type Action struct {
	Name ActionName
}

type ActionName int

const (
	GetRooms   ActionName = 0
	JoinRoom   ActionName = 1
	Post       ActionName = 2
	LeaveRoom  ActionName = 3
	CreateRoom ActionName = 4
)
