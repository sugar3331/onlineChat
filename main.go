package main

import (
	im "OnlineChat/handler"
	"OnlineChat/tools"
	"log"
	"net/http"
)

func main() {
	im.NewMysqlInit()
	tools.MongodbInit()
	http.HandleFunc("/publicChatHandler", im.PublicChatHandler)
	http.HandleFunc("/p2pChatHandler", im.P2PChatHandler)
	http.HandleFunc("/login", im.Login)
	http.HandleFunc("/register", im.Register)
	http.HandleFunc("/publicChat", im.PublicChat)
	http.HandleFunc("/p2pChat", im.P2pChat)
	log.Println("IM Websocket Starting server at: 9090...")
	err := http.ListenAndServe(":9090", nil)
	if err != nil {
		log.Fatal("聊天服务启动失败！！！:", err)
	}
}
