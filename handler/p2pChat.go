package im

import (
	"OnlineChat/tools"
	"github.com/gorilla/websocket"
	"html/template"
	"log"
	"net/http"
	"sync"
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
		//conn.Close()
		//p2pclients.Delete(userid) // 使用 Delete 方法来删除键值对，保证并发安全性
		//close(exitSignal)
	}()

	targetUserID := parseTargetUserIDFromURL(r)
	log.Println("身份验证完毕用户id: " + userid + " 用户: " + username + " 开始与对方id通信: " + targetUserID)
	go handleIncomingMessage(userid, targetUserID, username, conn, exitSignal)
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
			pup2pmessage := fullMessage{
				Message:  string(rep2pmessage),
				Username: username,
				Userid:   userid,
			}

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
