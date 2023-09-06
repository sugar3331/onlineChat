package im

import (
	"OnlineChat/tools"
	"context"
	"encoding/json"
	"github.com/gorilla/websocket"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"html/template"
	"log"
	"net/http"
	"sync"
	"time"
)

func P2pChat(w http.ResponseWriter, r *http.Request) {
	// 解析前端界面的HTML模板
	tmpl, err := template.ParseFiles("./static/p2pChat.html")
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

var p2pclients sync.Map

func P2PChatHandler(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	log.Println("开始进行点对点通信，首先身份验证")
	userid, username, err := tools.Token.ImJwtAuthMiddleware(token)
	if err != nil {
		return
	}

	targetUserID := parseTargetUserIDFromURL(r)
	if userid == targetUserID {
		w.WriteHeader(http.StatusBadRequest)
		w.Header().Set("Content-Type", "application/json")
		response := map[string]string{"error": "不能和自己聊天"}
		json.NewEncoder(w).Encode(response)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Failed to upgrade websocket:", err)
		return
	}

	//username := r.Context().Value("userName").(string)
	p2pclients.Store(userid, conn)
	// 连接关闭时，从映射中删除用户ID和连接
	exitSignal := make(chan struct{})
	defer func() {
		conn.Close()
		p2pclients.Delete(userid) // 使用 Delete 方法来删除键值对，保证并发安全性
		close(exitSignal)
	}()

	log.Println("身份验证完毕用户id: " + userid + " 用户: " + username + " 开始与对方id通信: " + targetUserID)
	go sendP2PRecentMessages(userid, targetUserID, conn)
	go handleIncomingMessage(userid, targetUserID, username, conn, exitSignal)

	// 在这里等待信号通知退出
	<-exitSignal
}

func handleIncomingMessage(userid, targetUserID, username string, conn *websocket.Conn, exitSignal <-chan struct{}) {
	for {
		select {
		case <-exitSignal:
			return
		default:
			_, rep2pmessage, err := conn.ReadMessage()
			if err != nil {
				log.Println("Failed to read message from websocket:", err)
				return
			}
			log.Println("Received message:", string(rep2pmessage))
			sendtimeNow := time.Now()
			pup2pmessage := fullMessage{
				Message:  string(rep2pmessage),
				Username: username,
				Userid:   userid,
				SendTime: sendtimeNow,
			}
			go func() {
				err := saveP2PMessageToMongoDB(userid, targetUserID, pup2pmessage)
				if err != nil {
					log.Println("mongodbp2p 存储消息有误 ", err)
				}
			}()
			// 转发消息给目标用户
			if targetConn, ok := p2pclients.Load(targetUserID); ok {
				err = targetConn.(*websocket.Conn).WriteJSON(pup2pmessage)
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
	return r.URL.Query().Get("taruserid")
}

func sendP2PRecentMessages(userid, targetUserID string, conn *websocket.Conn) {
	opts := options.Find().SetSort(bson.D{{"_id", -1}}).SetLimit(30)
	filter := bson.D{{}}
	collectionName := convert(userid, targetUserID)
	collection := tools.GolbalMogodb.Database("ImP2PChat").Collection(collectionName)
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
		//if targetConn, ok := p2pclients.Load(userid); ok {
		//	err = targetConn.(*websocket.Conn).WriteJSON(recentMessages[i])
		err = conn.WriteJSON(recentMessages[i])
		if err != nil {
			log.Println(err)
			break
		}

	}
}

func saveP2PMessageToMongoDB(userid, targetUserID string, msg fullMessage) error {
	collectionName := convert(userid, targetUserID)
	collection := tools.GolbalMogodb.Database("ImP2PChat").Collection(collectionName)
	_, err := collection.InsertOne(context.Background(), msg)
	if err != nil {
		return err
	}
	return nil
}

func convert(a, b string) string {
	if a > b {
		return a + "-" + b
	} else {
		return b + "-" + a
	}
}
