package main

import (
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

type Server struct {
	Ip   string
	Port int

	// 在线用户列表
	onlineMap map[string]*User
	mapLock   sync.RWMutex

	// 消息广播channel
	message chan string
}

func NewServer(ip string, port int) *Server {
	server := &Server{
		Ip:        ip,
		Port:      port,
		onlineMap: make(map[string]*User),
		message:   make(chan string),
	}
	return server
}

func (server *Server) Broadcast(user *User, msg string) {
	message := "[" + user.Addr + "] " + user.Name + " : " + msg
	server.message <- message
}

func (server *Server) listenBroadcast() {
	for {
		msg := <-server.message
		server.mapLock.Lock()
		for _, user := range server.onlineMap {
			user.C <- msg
		}
		server.mapLock.Unlock()
	}
}

func (server *Server) handler(conn net.Conn) {
	fmt.Println("连接建立成功")
	user := NewUser(conn, server)

	user.Online()

	isLive := make(chan bool)

	// 接受客户端数据
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := user.conn.Read(buf)
			if n == 0 {
				user.Offline()
				return
			}
			if err != nil && err != io.EOF {
				fmt.Println("Conn read err: ", err)
				return
			}
			msg := string(buf[:n-1])
			user.DoMessage(msg)
			isLive <- true
		}
	}()

	for {
		select {
		case <-isLive:

		case <-time.After(time.Second * 60):
			// 将当前的用户关闭
			user.conn.Write([]byte(user.Name + "被强踢下线\n"))
			close(user.C)
			user.conn.Close()
			return
		}
	}
}

func (server *Server) Start() {
	// socket listen
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", server.Ip, server.Port))
	if err != nil {
		fmt.Println("net.Listen err:", err)
		return
	}
	// close socket
	defer listener.Close()

	go server.listenBroadcast()

	for {
		// accept connections
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("net.Accept err: ", err)
			continue
		}
		// do handler
		go server.handler(conn)
	}
}
