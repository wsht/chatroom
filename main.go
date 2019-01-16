package main

import (
	"fmt"
	"os"
	"socket/chatroom/backend"
	"socket/chatroom/chat"
	"strconv"
)

func main() {
	types := os.Args[1]
	switch types {
	case "server":
		chat.TestServer()
	case "client":
		chat.TestClient(os.Args[2])
	case "multiserver":
		nums, err := strconv.Atoi(os.Args[2])
		if err != nil {
			fmt.Println(err)
			return
		}
		chat.TestMultiClient(nums)
	case "backend":
		backend.BackEndServer()
	}
}
