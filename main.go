package main

import (
	im "OnlineChat/handler"
	"log"
	"net/http"
)

func main() {
	im.NewMysqlInit()
	http.HandleFunc("/publicChat", im.HandIeWebSocket)
	http.HandleFunc("/p2pChat", im.HandleP2PChat)
	http.HandleFunc("/login", im.Login)
	http.HandleFunc("/chat", im.HandleChat)
	log.Println("IM Websocket Starting server at: 9090...")
	err := http.ListenAndServe(":9090", nil)
	if err != nil {
		log.Fatal("聊天服务启动失败！！！:", err)
	}
}
