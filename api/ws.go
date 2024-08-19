package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/vercel/go-bridge/go/bridge"
)

// 定义 WebSocket 升级器
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许所有来源
	},
}

// 使用 sync.Map 来管理用户连接
var clients sync.Map

// 消息结构体
type Message struct {
	TargetUserId string `json:"targetUserId"`
	Content      string `json:"content"`
}

// 处理客户端连接
func handleConnections(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatalf("Failed to upgrade connection: %v", err)
		return
	}

	// 获取 userId
	userId := r.URL.Path[len("/api/ws/"):]

	// 将连接存储到 clients map 中
	clients.Store(userId, conn)
	fmt.Printf("Client %s connected\n", userId)

	// 监听并转发消息
	for {
		_, messageData, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Error reading message from %s: %v", userId, err)
			break
		}

		// 解析消息
		var msg Message
		err = json.Unmarshal(messageData, &msg)
		if err != nil {
			log.Printf("Invalid message format from %s: %v", userId, err)
			continue
		}

		fmt.Printf("Message from %s to %s: %s\n", userId, msg.TargetUserId, msg.Content)

		// 查找目标用户的连接并转发消息
		if targetConn, ok := clients.Load(msg.TargetUserId); ok {
			if err := targetConn.(*websocket.Conn).WriteMessage(websocket.TextMessage, messageData); err != nil {
				log.Printf("Error sending message to %s: %v", msg.TargetUserId, err)
			}
		} else {
			log.Printf("Target user %s not found", msg.TargetUserId)
		}
	}

	// 客户端断开连接后关闭 WebSocket 并移除连接
	conn.Close()
	clients.Delete(userId)
	fmt.Printf("Client %s disconnected\n", userId)
}

// Lambda 入口函数
func Handler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/api/ws/" || len(r.URL.Path) > len("/api/ws/") {
		handleConnections(w, r)
	} else {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "Endpoint not found")
	}
}

func main() {
	bridge.Start(Handler)
}

