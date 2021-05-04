package chat

import "fmt"

func LogError(formatStr string, err error) {
	e := fmt.Errorf(formatStr, err)
	fmt.Println(e.Error())
}
