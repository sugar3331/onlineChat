package im

import (
	"OnlineChat/tools"
	"context"
	"fmt"
	"github.com/gorilla/websocket"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"html/template"
	"log"
	"net/http"
	"sync"
	"time"
)

type Client struct {
	conn     *websocket.Conn
	username string
	userid   string
}

var clients = &sync.Map{}

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
	Userid   string
	SendTime time.Time
}

func PublicChat(w http.ResponseWriter, r *http.Request) {
	// 解析前端界面的HTML模板
	tmpl, err := template.ParseFiles("./static/chat.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 渲染界面模板，并将其写入响应中
	err = tmpl.Execute(w, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// HandIeWebSocket 客户端连接到公共聊天室接口
func PublicChatHandler(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	userid, username, err := tools.Token.ImJwtAuthMiddleware(token)
	if err != nil {
		return
	}
	//升级http连接为WebSocket连接
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println(err)
		return
	}

	// 创建 Client 结构体，并将连接和用户名保存其中
	client := &Client{
		conn:     conn,
		userid:   userid,
		username: username,
	}

	// 将连接添加到clients列表
	clients.Store(client, true)
	defer func() {
		conn.Close()
		// 连接断开时从clients列表移除该连接
		clients.Delete(client)
	}()
	//从mongodb里面读缓存
	go sendRecentMessages(client)
	// 处理WebSocket连接
	for {
		// 读取消息
		messageType, msg, err := conn.ReadMessage()
		if err != nil {
			log.Println(err)
			break
		}
		log.Println("Received message:", string(msg))
		sendtimeNow := time.Now()
		fullMe := fullMessage{
			Message:  string(msg),
			Username: username,
			Userid:   userid,
			SendTime: sendtimeNow,
		}
		log.Println("开始发消息，用户id: " + userid + ":" + string(msg))
		go saveMessageToMongoDB(fullMe)
		// 广播消息给所有客户端
		go broadcastMessage(messageType, fullMe, client)
	}
}

func sendRecentMessages(client *Client) {
	opts := options.Find().SetSort(bson.D{{"_id", -1}}).SetLimit(30)
	filter := bson.D{{}}
	collection := tools.GolbalMogodb.Database("ImChat").Collection("publicChat")
	cursor, err := collection.Find(context.TODO(), filter, opts)
	if err != nil {
		log.Println(err)
		return
	}

	var recentMessages []fullMessage
	for cursor.Next(context.TODO()) {
		var message fullMessage
		if err := cursor.Decode(&message); err != nil {
			log.Println(err)
			break
		}
		recentMessages = append(recentMessages, message)
	}

	// 倒序发送最近的 30 条记录给客户端
	for i := len(recentMessages) - 1; i >= 0; i-- {
		err := client.conn.WriteJSON(recentMessages[i])
		if err != nil {
			log.Println(err)
			break
		}
	}
}

func saveMessageToMongoDB(msg fullMessage) error {
	collection := tools.GolbalMogodb.Database("ImChat").Collection("publicChat")
	_, err := collection.InsertOne(context.Background(), msg)
	if err != nil {
		return err
	}
	return nil
}

// broadcastMessage 服务端把用户发送的消息推送给所有在线用户的广播函数
func broadcastMessage(messageType int, fullMe fullMessage, sender *Client) {
	clients.Range(func(key, value interface{}) bool {
		client := key.(*Client)

		// 不向消息发送者推送消息
		if client == sender {
			return true
		}

		err := client.conn.WriteJSON(fullMe)
		if err != nil {
			log.Println(err)
			client.conn.Close()
			clients.Delete(client)
		}
		return true
	})
}
