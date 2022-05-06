package main

import (
	"fmt"
	"net"
	"strings"
)

type User struct {
	Name   string
	Addr   string
	C      chan string
	conn   net.Conn
	server *Server
}

func NewUser(conn net.Conn, server *Server) *User {
	user := &User{
		Name:   conn.RemoteAddr().String(),
		Addr:   conn.RemoteAddr().String(),
		C:      make(chan string),
		conn:   conn,
		server: server,
	}
	go user.ListenMessage()
	return user
}

func (u *User) ListenMessage() {
	for {
		msg := <-u.C
		u.conn.Write([]byte(msg + "\n"))
	}
}

func (u *User) Online() {
	u.server.mapLock.Lock()
	u.server.onlineMap[u.Name] = u
	u.server.mapLock.Unlock()
	// 广播用户上线消息
	u.server.Broadcast(u, "上线了")
}

func (u *User) Offline() {
	u.server.mapLock.Lock()
	delete(u.server.onlineMap, u.Name)
	u.server.mapLock.Unlock()
	// 广播用户下线消息
	u.server.Broadcast(u, "下线了")
}

func (u *User) DoMessage(msg string) {
	fmt.Println(msg)
	if msg == "who" {
		u.server.mapLock.Lock()
		for _, user := range u.server.onlineMap {
			onlineMsg := "[" + user.Addr + "] " + user.Name + " 在线......"
			u.conn.Write([]byte(onlineMsg + "\n"))
		}
		u.server.mapLock.Unlock()
	} else if len(msg) > 7 && msg[:6] == "rename" {
		newName := strings.Split(msg, "|")[1]
		_, ok := u.server.onlineMap[newName]
		if ok {
			u.conn.Write([]byte("当前用户名已经被使用\n"))
		} else {
			u.server.mapLock.Lock()
			delete(u.server.onlineMap, u.Name)
			u.server.onlineMap[newName] = u
			u.server.mapLock.Unlock()
			u.Name = newName
			u.conn.Write([]byte("修改用户名成功"))
		}
	} else if len(msg) > 4 && msg[:3] == "to|" {
		args := strings.Split(msg, "|")
		if len(args) != 3 {
			u.conn.Write([]byte("消息格式有误，请重新输入\n"))
			return
		}
		target := args[1]
		content := args[2]
		if target == "" {
			u.conn.Write([]byte("对方用户名不能为空\n"))
			return
		}
		_, ok := u.server.onlineMap[target]
		if !ok {
			u.conn.Write([]byte("对方用户不存在\n"))
			return
		}
		if content == "" {
			u.conn.Write([]byte("发送消息不能为空\n"))
			return
		}
		u.server.onlineMap[target].conn.Write([]byte(u.Name + " 对你说: " + content + "\n"))
	} else {
		u.server.Broadcast(u, msg)
	}
}
