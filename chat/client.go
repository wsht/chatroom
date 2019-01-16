package chat

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"socket/chatroom/smsg"
	"strconv"
	"sync"
	"time"
)

var isSilence bool

func handleReadedMsg(buf []byte) {
	// if isSilence {
	// 	return
	// }
	a := smsg.InitMsg(string(buf))
	switch a.GetCommand() {
	case "result":
		// fmt.Println(a.GetCommand())
		// fmt.Println(a.(*smsg.SocketMsg).Content)
	case "receivemessage":
		// fmt.Println(a.GetCommand())
		if isSilence {
			return
		}
		if a.(*smsg.SocketMsg).Enc == "yes" {
			// fmt.Println(a.(*smsg.SocketMsg).SendTimer)
			startTimer, err := time.Parse(time.RFC3339Nano, a.(*smsg.SocketMsg).SendTimer)
			if err != nil {
				fmt.Println("get start timer error", err)
				return
			}
			fmt.Println("receive delay is ", time.Now().Sub(startTimer).String())
		} else {
			// fmt.Println(a.(*smsg.SocketMsg).Content)
		}
	case "error":
		fmt.Println(a.(*smsg.SocketMsg).Content)
	}
}

func TestClient(uid string) {
	localAddr := &net.TCPAddr{IP: net.IPv4zero, Port: 0}
	remoteAddr := &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 9981}

	conn, err := net.DialTCP("tcp", localAddr, remoteAddr)
	if err != nil {
		fmt.Println(err)
		return
	}

	defer conn.Close()

	//login
	fmt.Printf("user uid %s request login\n", uid)
	msg := smsg.LoginReqMsg{
		Command: "login",
		Uid:     uid,
		Encpass: "xxxaaaa",
	}
	// fmt.Printf("send msg is %s \n", msg.String())

	_, err = conn.Write([]byte(msg.String()))
	if err != nil {
		fmt.Println("send login error with err", err)
	}
	wg := sync.WaitGroup{}

	send := make(chan []byte, 100)
	//send
	go func() {
		for b := range send {
			conn.Write(b)
		}
	}()

	//心跳机制
	go func() {
		t := time.Tick(10 * time.Second)
		for {
			select {
			case _ = <-t:
				fmt.Println("send ttl msg")
				msg := &smsg.SocketMsg{
					Command: "sendmessage",
					Enc:     "no",
					Content: "heartbeat",
				}
				send <- []byte(msg.String())
			}
		}
	}()

	go func() {
		rand.Seed(time.Now().UnixNano())
		for {
			n := rand.Intn(5) + 5
			time.Sleep(time.Duration(n) * time.Second)
			//先不发消息
			continue
			content := map[string]string{}
			content["uid"] = uid
			content["t"] = "101"
			content["msg"] = "i'm very happy to join faimly"
			contentstr, err := json.Marshal(content)
			if err != nil {
				conn.Close()
				return
			}
			contentstr2 := smsg.ContentEncode(string(contentstr))
			msg := &smsg.SocketMsg{
				Command:   "sendmessage",
				Enc:       "yes",
				Content:   contentstr2,
				SendTimer: time.Now().Format(time.RFC3339Nano),
			}
			// fmt.Println("send time is", msg.SendTimer)
			send <- []byte(msg.String())
		}
	}()

	//read
	wg.Add(1)
	go func() {
		defer fmt.Println("run end")
		defer wg.Done()
		buf := make([]byte, 1024)
		var n int
		var err error
		var readedIndex int
		var msgLens int
		var searchIndex int

		message := make([]byte, 1024)
		endLineByte := []byte("\r\n")
		unreadBufLens := 0
		for {
			// fmt.Println("start read from socket")
			n, err = conn.Read(message)
			if err != nil {
				fmt.Println("read error", err)
				conn.Close()
				return
			}
			readedIndex = 0
			for readedIndex < n {
				if msgLens == 0 {
					searchIndex = bytes.Index(message[readedIndex:], endLineByte)
					if searchIndex == -1 {
						buf = append(buf, message[readedIndex:]...)
						readedIndex += n
						continue
					}

					//出现换行符号
					//将换行符号倒入到缓冲buf中
					//这时候，如果buf中包含content-length，那么他就是content-length
					buf = append(buf, message[readedIndex:readedIndex+searchIndex]...)
					msgLens, err = strconv.Atoi(string(buf))
					buf = []byte{}
					readedIndex += searchIndex + 2
					if err != nil {
						msgLens = 0
						continue
					}
				}

				if msgLens == 0 && n-searchIndex <= 0 {
					break
				}

				if msgLens > 0 {
					// fmt.Println("msglens is", msgLens, n)
					if msgLens-len(buf)-(n-readedIndex) > 0 {
						buf = append(buf, message[readedIndex:]...)
						break
					} else if msgLens-len(buf)-(n-readedIndex) == 0 {
						// fmt.Println("just ok")
						buf = append(buf, message[readedIndex:]...)
						handleReadedMsg(buf)
						buf = []byte{}
						msgLens = 0
						break
					} else {
						// fmt.Println("overflow")
						unreadBufLens = msgLens - len(buf)
						// fmt.Println(readedIndex, msgLens, unreadBufLens)
						buf = append(buf, message[readedIndex:readedIndex+unreadBufLens]...)
						handleReadedMsg(buf)
						buf = []byte{}
						msgLens = 0
						readedIndex += unreadBufLens
					}
				}
			}
		}
	}()

	wg.Wait()
}

// 37
// command=login
// encpass=xxxaaaa
// uid=2

func TestMultiClient(nums int) {
	isSilence = true
	wg := sync.WaitGroup{}
	for i := 1; i <= nums; i++ {
		uid := strconv.Itoa(i)
		wg.Add(1)
		go func() {
			defer wg.Done()
			TestClient(uid)
		}()
		time.Sleep(10 * time.Millisecond)
	}
	wg.Wait()
}
