package backend

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"socket/chatroom/smsg"

	"github.com/garyburd/redigo/redis"
)

type user struct {
	uid    int
	server *net.TCPAddr
	client *net.TCPAddr
}

type multiClient map[string]*user

type room map[int]*multiClient

//in this server we use redis to store the user info
//and we used http server to communicate to the socket

var rconn redis.Conn

var chatserver = []string{"127.0.0.1:9980"}

func userSocketMix(uid, serveraddr, clientaddr string) string {
	h := md5.New()
	h.Write([]byte(uid + serveraddr + clientaddr))
	cip := h.Sum(nil)
	return hex.EncodeToString(cip)
}

func userSocketInfoJson(uid, serveraddr, clientaddr string) string {
	detailInfo := map[string]string{}
	detailInfo["uid"] = uid
	detailInfo["serveraddr"] = serveraddr
	detailInfo["clientaddr"] = clientaddr
	str, err := json.Marshal(detailInfo)
	if err != nil {
		return ""
	}
	return string(str)
}

func loginSuccessNotice(uid string) {

	content := map[string]string{}
	content["t"] = "101"

	content["msg"] = fmt.Sprintf("user uid %s enter room", uid)

	jsonstr, err := json.Marshal(content)
	if err != nil {
		return
	}

	contentstr := smsg.ContentEncode(string(jsonstr))
	enc := "yes"
	sendRoom(contentstr, enc, "")
}

func sendRoom(content string, enc string, sendtimer string) {
	params := url.Values{}
	params.Add("content", content)
	params.Add("encrypted", enc)
	if sendtimer != "" {
		params.Add("sendtimer", sendtimer)
	}
	for i := 0; i < len(chatserver); i++ {
		rurl := fmt.Sprintf("http://%s/sendroom?", chatserver[i])
		rurl += params.Encode()
		//注意 这里的http.Get(url) 返回一个resp;
		//每个resp 都是单独的一个client,需要及时关闭。否则句柄会一直占用，
		//导致 too many open file

		// fmt.Println(rurl)
		resp, err := http.Get(rurl)
		if err != nil {
			fmt.Println("error http get", err)
			return
		}

		defer resp.Body.Close()
		_, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Println("get body error is", err)
			return
		}
		// fmt.Println(string(body))
	}
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	uid := r.FormValue("uid")
	serveraddr := r.FormValue("serveraddr")
	clientaddr := r.FormValue("clientaddr")
	// fmt.Println(uid, serveraddr, clientaddr)
	if uid == "" || serveraddr == "" || clientaddr == "" {
		fmt.Fprintf(w, "-3001")
		return
	}
	mix := userSocketMix(uid, serveraddr, clientaddr)
	rconn.Do("sadd", "uid_"+uid, mix)

	dinfo := userSocketInfoJson(uid, serveraddr, clientaddr)
	rconn.Do("set", mix, dinfo)
	loginSuccessNotice(uid)
	fmt.Fprintf(w, "+ok")
}

func sendMsgHandler(w http.ResponseWriter, r *http.Request) {
	uid := r.FormValue("uid")
	serveraddr := r.FormValue("serveraddr")
	clientaddr := r.FormValue("clientaddr")
	enc := r.FormValue("enc")
	content := r.FormValue("content")
	sendtimer := r.FormValue("sendtimer")
	if uid == "" || serveraddr == "" || clientaddr == "" || enc == "" || content == "" {
		fmt.Fprintf(w, "-3001")
		return
	}
	if enc == "yes" {
		content = smsg.ContentDecode(content)
	}

	if content == "heartbeat" {
		fmt.Fprintf(w, "+ok")
		return
	}
	if enc == "yes" {
		content = smsg.ContentEncode(content)
	}
	sendRoom(content, enc, sendtimer)
	fmt.Fprintf(w, "+ok")
}

func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	uid := r.FormValue("uid")
	serveraddr := r.FormValue("serveraddr")
	clientaddr := r.FormValue("clientaddr")
	if uid == "" || serveraddr == "" || clientaddr == "" {
		fmt.Fprintf(w, "-3001")
	}
	mix := userSocketMix(uid, serveraddr, clientaddr)
	rconn.Do("smove", "uid_"+uid, mix)
	rconn.Do("del", mix)

	fmt.Fprintf(w, "+ok")
}

func BackEndServer() {
	conn, err := redis.Dial("tcp", "127.0.0.1:6379")
	if err != nil {
		fmt.Println(err)
		return
	}
	rconn = conn
	http.HandleFunc("/login", LoginHandler)
	http.HandleFunc("/sendmsg", sendMsgHandler)
	http.ListenAndServe(":8081", nil)

}
