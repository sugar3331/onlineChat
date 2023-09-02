package im

import (
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"strconv"
	"sync"
)

type Client struct {
	conn     *websocket.Conn
	username string
	userid   int64
}

var clients = &sync.Map{}
var p2pclients = sync.Map{}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type fullMessage struct {
	Message  string
	Username string
	Userid   int64
}

// HandIeWebSocket 客户端连接到公共聊天室接口
func HandIeWebSocket(w http.ResponseWriter, r *http.Request) {
	//升级http连接为WebSocket连接
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer conn.Close()

	userid := r.Context().Value("userID").(int64)
	username := r.Context().Value("userName").(string)

	// 创建 Client 结构体，并将连接和用户名保存其中
	client := &Client{
		conn:     conn,
		userid:   userid,
		username: username,
	}
	// 将连接添加到clients列表
	clients.Store(client, true)

	// 处理WebSocket连接
	for {
		// 读取消息
		messageType, msg, err := conn.ReadMessage()
		if err != nil {
			log.Println(err)
			break
		}
		log.Println("Received message:", string(msg))
		fullMe := fullMessage{
			Message:  string(msg),
			Username: username,
			Userid:   userid,
		}
		// 广播消息给所有客户端
		go broadcastMessage(messageType, fullMe, client)
	}

	// 连接断开时从clients列表移除该连接
	clients.Delete(client)

}

// broadcastMessage 服务端把用户发送的消息推送给所有在线用户的广播函数
func broadcastMessage(messageType int, fullMe fullMessage, sender *Client) {
	clients.Range(func(key, value interface{}) bool {
		client := key.(*Client)
		err := client.conn.WriteJSON(fullMe)
		if err != nil {
			log.Println(err)
			client.conn.Close()
			clients.Delete(client)
		}
		return true
	})
}

func HandleP2PChat(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Failed to upgrade websocket:", err)
		return
	}

	userid := r.Context().Value("userID").(int64)
	//username := r.Context().Value("userName").(string)
	p2pclients.Store(userid, conn)
	// 连接关闭时，从映射中删除用户ID和连接
	exitSignal := make(chan struct{})
	defer func() {
		conn.Close()
		p2pclients.Delete(userid) // 使用 Delete 方法来删除键值对，保证并发安全性
		close(exitSignal)
	}()

	targetUserIDStr := parseTargetUserIDFromURL(r)
	targetUserID, _ := strconv.ParseInt(targetUserIDStr, 10, 64)
	go handleIncomingMessage(targetUserID, conn, exitSignal)
}

func handleIncomingMessage(userID int64, conn *websocket.Conn, exitSignal <-chan struct{}) {
	for {
		select {
		case <-exitSignal:
			return
		default:
			_, message, err := conn.ReadMessage()
			if err != nil {
				log.Println("Failed to read message from websocket:", err)
				break
			}

			// 转发消息给目标用户
			if targetConn, ok := p2pclients.Load(userID); ok {
				err = targetConn.(*websocket.Conn).WriteJSON(message)
				if err != nil {
					log.Println("Failed to send message to target user:", err)
				}
			} else {
				log.Println("Target user is not connected.")
			}
		}
	}
}

func parseTargetUserIDFromURL(r *http.Request) string {
	return r.URL.Query().Get("Userid")
}
