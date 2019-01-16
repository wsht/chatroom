package smsg

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io/ioutil"
	"strings"
)

func ContentEncode(msg string) string {
	var b bytes.Buffer
	w := gzip.NewWriter(&b)
	w.Write([]byte(msg))
	w.Flush()
	defer w.Close()

	return b.String()
}

func ContentDecode(msggz string) string {
	sr := strings.NewReader(msggz)
	gzr, err := gzip.NewReader(sr)
	if err != nil {
		return ""
	}
	defer gzr.Close()
	undata, err := ioutil.ReadAll(gzr)

	return string(undata)
}

type MsgInterface interface {
	Init(msg string)
	MsgLen() int
	String() string
	GetCommand() string
}

func feild(str, key string) string {
	start := strings.Index(str, key)
	if start == -1 {
		return ""
	}
	start += len(key)
	end := strings.Index(str[start:], "\r\n")
	if end == -1 {
		return ""
	}
	end += start
	// fmt.Println(str[start:end], start, end)

	return str[start:end]
}

func InitMsg(str string) MsgInterface {
	var a MsgInterface

	command := feild(str, "command=")
	if command == "" {
		return nil
	}
	// fmt.Println("command is", command)
	switch command {
	case "login":
		a = &LoginReqMsg{}
		// fmt.Println("a to login req msg", a.(*LoginReqMsg))
	case "receivemessage":
		fallthrough
	case "sendmessage":
		fallthrough
	case "result":
		fallthrough
	case "error":
		a = &SocketMsg{}

	default:
		return nil
	}
	a.Init(str)
	// fmt.Println(a)
	return a
}

type SocketMsg struct {
	Command   string
	Enc       string
	Content   string
	SendTimer string
}

func (sm *SocketMsg) Init(msg string) {
	sm.Command = feild(msg, "command=")
	sm.Enc = feild(msg, "enc=")
	sm.Content = feild(msg, "content=")
	sm.SendTimer = feild(msg, "sendtimer=")
}
func (sm *SocketMsg) GetCommand() string {
	return sm.Command
}
func (sm *SocketMsg) MsgLen() int {
	return len("command=") + len(sm.Command) + 2 + len("enc=") + len(sm.Enc) + 2 + len("content=") + len(sm.Content) + 2 + len("sendtimer=") + len(sm.SendTimer) + 2
}

func (sm *SocketMsg) String() string {
	return fmt.Sprintf("%d\r\n%s\r\n%s\r\n%s\r\n%s\r\n", sm.MsgLen(), "command="+sm.Command, "enc="+sm.Enc, "content="+sm.Content, "sendtimer="+sm.SendTimer)
}

type LoginReqMsg struct {
	Command string
	Encpass string
	Uid     string
}

func (lrm *LoginReqMsg) Init(msg string) {
	lrm.Command = feild(msg, "command=")
	lrm.Encpass = feild(msg, "encpass=")
	lrm.Uid = feild(msg, "uid=")
}

func (lrm *LoginReqMsg) GetCommand() string {
	return lrm.Command
}

func (lrm *LoginReqMsg) MsgLen() int {
	return len("command=") + len(lrm.Command) + 2 + len("encpass=") + len(lrm.Encpass) + 2 + len("uid=") + len(lrm.Uid) + 2
}

func (lrm *LoginReqMsg) String() string {
	return fmt.Sprintf("%d\r\n%s\r\n%s\r\n%s\r\n", lrm.MsgLen(), "command="+lrm.Command, "encpass="+lrm.Encpass, "uid="+lrm.Uid)
}
