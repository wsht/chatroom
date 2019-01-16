package chat

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"socket/chatroom/smsg"
	"strconv"
	"sync"
	"time"
)

// todo 心跳包以及退出通知
func buildUuid(saddr, clientaddr string) string {
	h := md5.New()
	h.Write([]byte(saddr + clientaddr))
	cip := h.Sum(nil)
	return hex.EncodeToString(cip)
}

type ChatClient struct {
	uid       string
	connected bool
	fd        *net.TCPConn
	addr      *net.TCPAddr
	ttl       int //心跳包主动断开的依据
	uuid      string
	data      chan []byte
}

func (c *ChatClient) getUuid() string {
	// h := md5.New()
	// //strconv.Itoa(c.uid)
	// h.Write([]byte(saddr.String() + c.addr.String()))
	// cip := h.Sum(nil)
	// return hex.EncodeToString(cip)
	return c.uuid
}

func (c *ChatClient) setuuid(saddr *net.TCPAddr) {
	c.uuid = buildUuid(saddr.String(), c.addr.String())
}

// type clientMap sync.Map

type ChatServer struct {
	members   *sync.Map
	count     int
	listener  *net.TCPListener
	addr      *net.TCPAddr
	income    chan *ChatClient
	outcome   chan *ChatClient
	broadcase chan []byte
	heartbeat chan *ChatClient
}

func (s *ChatServer) init() {
	for {
		select {
		case c := <-s.income:
			s.count++
			c.connected = true
		case c := <-s.outcome:
			s.members.Delete(c.getUuid())
			if c.connected != false {
				s.count--
				close(c.data)
				c.connected = false
			}
		case c := <-s.heartbeat:
			//这里是设置为0了，但是还是没有起作用
			c.ttl = 0
		case message := <-s.broadcase:
			//这里能不能作为做个队列来做这件事情呢
			//cal cost time
			start := time.Now()
			s.members.Range(func(uuid, client interface{}) bool {
				//这里是不是需要设置一个超时
				// _, err := client.(*ChatClient).fd.Write(message)
				// if err != nil {
				// 	//client out
				// 	s.outcome <- client.(*ChatClient)
				// }
				select {
				case client.(*ChatClient).data <- message:
					{

					}
				default:
					fmt.Println(uuid, "out of room")
					s.outcome <- client.(*ChatClient)
					// close(client.(*ChatClient).data)
					// s.count--
					// s.members.Delete(uuid)
				}
				return true
			})

			fmt.Println("broadcase cost const time:", time.Now().Sub(start))
		}
	}
}

func (s *ChatServer) receive(c *ChatClient) {
	buf := []byte{}
	//为了不频繁的创建变量 这里提前声明
	var n int
	var err error
	var readedIndex int //记录当前接受消息轮训的角标
	var msgLens int     //一条完整消息的长度
	var searchIndex int
	message := make([]byte, 1024)
	endLineByte := []byte("\r\n")
	unreadBufLens := 0
	for {
		//注意这里
		// message := make([]byte, 15)
		n, err = c.fd.Read(message)
		if err != nil {
			s.outcome <- c
			c.fd.Close()
			return
		}
		readedIndex = 0
		// fmt.Println("recv msg is", string(message))
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
					s.messageHandler(buf, c)
					buf = []byte{}
					msgLens = 0
					break
				} else {
					// fmt.Println("overflow")
					unreadBufLens = msgLens - len(buf)
					// fmt.Println(readedIndex, msgLens, unreadBufLens)
					buf = append(buf, message[readedIndex:readedIndex+unreadBufLens]...)
					s.messageHandler(buf, c)
					buf = []byte{}
					msgLens = 0
					readedIndex += unreadBufLens
				}
			}
		}
	}
}

func (s *ChatServer) send(c *ChatClient) {
	for {
		select {
		case message, ok := <-c.data:
			if !ok {
				// fmt.Println("send data get error")
				// s.outcome <- c
				// fmt.Println(c.connected)
				c.fd.Close()
				return
			}
			// fmt.Println(len(c.data))
			c.fd.Write(message)
		}
	}
}

func (s *ChatServer) messageHandler(msg []byte, c *ChatClient) {
	// fmt.Println("messagehandler", string(msg))
	a := smsg.InitMsg(string(msg))
	switch a.GetCommand() {
	case "login":
		if loginReq(a.(*smsg.LoginReqMsg), s, c) {
			r := &smsg.SocketMsg{
				Command: "result",
				Enc:     "no",
				Content: "login.success",
			}
			c.uid = a.(*smsg.LoginReqMsg).Uid
			c.data <- []byte(r.String())
			//todo err notice
		} else {
			fmt.Printf("user uid %s login failed \n", a.(*smsg.LoginReqMsg).Uid)
			r := &smsg.SocketMsg{
				Command: "result",
				Enc:     "no",
				Content: "login.failed",
			}
			c.data <- []byte(r.String())
		}
	case "sendmessage":
		//todo sendmessage event
		if sendMsgReq(a.(*smsg.SocketMsg), s, c) {
			r := &smsg.SocketMsg{
				Command: "result",
				Enc:     "no",
				Content: "sendmsg.success",
			}
			c.data <- []byte(r.String())
			//心跳包处理
			if a.(*smsg.SocketMsg).Content == "heartbeat" {
				fmt.Println("receive heart beat from", c.uid)
				s.heartbeat <- c
			}
		} else {
			r := &smsg.SocketMsg{
				Command: "result",
				Enc:     "no",
				Content: "sendmsg.failed",
			}
			c.data <- []byte(r.String())
		}
	}
}

func (s *ChatServer) memberCount() {
	c := time.Tick(10 * time.Second)
	for {
		select {
		case _ = <-c:
			fmt.Println("current member count is:", s.count)
		}
	}
}

func (s *ChatServer) heartBeat() {
	t := time.Tick(10 * time.Second)
	for {
		select {
		case _ = <-t:
			s.members.Range(func(uuid, client interface{}) bool {
				fmt.Println(client.(*ChatClient).ttl)
				if client.(*ChatClient).ttl > 2 {
					//todo out of room
					//这里是不是应该做一个像客户端发送消息，如果发送不成功，则不需要关闭
					msg := &smsg.SocketMsg{
						Command: "error",
						Enc:     "no",
						Content: "logout",
					}
					_, err := client.(*ChatClient).fd.Write([]byte(msg.String()))
					if err == nil {
						client.(*ChatClient).fd.Close()
					}
					s.outcome <- client.(*ChatClient)
				}
				fmt.Println(client.(*ChatClient).uid, "ttl add 1")
				client.(*ChatClient).ttl += 1
				return true
			})
		}
	}
}

func (s *ChatServer) httpServer() {
	http.HandleFunc("/sendroom", func(w http.ResponseWriter, r *http.Request) {
		content := r.FormValue("content")
		encrypted := r.FormValue("encrypted")
		sendtimer := r.FormValue("sendtimer")
		// fmt.Println("sendtimer is", sendtimer)
		if encrypted != "yes" && encrypted != "no" {
			w.Write([]byte("error"))
			return
		}

		msg := &smsg.SocketMsg{
			Command:   "receivemessage",
			Enc:       encrypted,
			Content:   content,
			SendTimer: sendtimer,
		}
		s.broadcase <- []byte(msg.String())
		w.Write([]byte("+OK"))
	})

	log.Fatal(http.ListenAndServe(":9980", nil))
}

//注意 golang里面 对于用户态来说，read是堵塞的，所以需要放在coronute
func (s *ChatServer) start() {
	fmt.Println("running")
	go s.httpServer()
	go s.memberCount()
	go s.init()
	go s.heartBeat()
	for {
		conn, err := s.listener.AcceptTCP()
		if err != nil {
			//handle error
			continue
		}
		addr, err := net.ResolveTCPAddr("tcp", conn.RemoteAddr().String())
		c := &ChatClient{fd: conn, addr: addr, data: make(chan []byte, 4096)}
		c.setuuid(s.addr)
		s.members.Store(c.getUuid(), c)
		//每一条保持一个协程
		// fmt.Println(c.addr.String() + " enter room")
		s.income <- c
		go s.send(c)
		go s.receive(c)
	}
}

func TestServer() {
	addr := &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 9981}
	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer listener.Close()
	n := 100
	s := ChatServer{
		members:   &sync.Map{},
		addr:      addr,
		listener:  listener,
		income:    make(chan *ChatClient, n),
		outcome:   make(chan *ChatClient, n),
		broadcase: make(chan []byte, 4096),
		heartbeat: make(chan *ChatClient, n),
	}
	s.start()
}

func loginReq(msg *smsg.LoginReqMsg, s *ChatServer, c *ChatClient) bool {
	url := "http://127.0.0.1:8081/login?"
	// fmt.Println("user uid ", msg.Uid, "request login")
	url += fmt.Sprintf("uid=%s&serveraddr=%s&clientaddr=%s", msg.Uid, s.addr.String(), c.addr.String())
	// fmt.Println("login req url is", url)
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("error http get", err)
		return false
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("get body error is", err)
		return false
	}
	// fmt.Println(string(body))
	if bytes.Equal(body, []byte("+ok")) {
		return true
	}
	return false
}

func sendMsgReq(msg *smsg.SocketMsg, s *ChatServer, c *ChatClient) bool {
	rurl := "http://127.0.0.1:8081/sendmsg?"
	params := url.Values{}

	params.Add("uid", c.uid)
	params.Add("serveraddr", s.addr.String())
	params.Add("clientaddr", c.addr.String())
	params.Add("enc", msg.Enc)
	params.Add("content", msg.Content)
	params.Add("sendtimer", msg.SendTimer)
	// url += fmt.Sprintf("uid=%s&serveraddr=%s&clientaddr=%s&enc=%s&content=%s&sendtimer=%s", c.uid, s.addr.String(), c.addr.String(), msg.Enc, msg.Content, msg.SendTimer)
	rurl += params.Encode()
	resp, err := http.Get(rurl)
	if err != nil {
		fmt.Println("error http get", err)
		return false
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("get body error is", err)
		return false
	}

	// fmt.Println(string(body))
	if bytes.Equal(body, []byte("+ok")) {
		return true
	}
	return false
}
