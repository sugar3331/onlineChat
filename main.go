package main

import (
	im "OnlineChat/handler"
	"OnlineChat/tools"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/publicChat", tools.Token.ImJwtAuthMiddleware(im.HandIeWebSocket))
	http.HandleFunc("/p2pChat", tools.Token.ImJwtAuthMiddleware(im.HandleP2PChat))
	log.Println("IM Websocket Starting server at: 9090...")
	err := http.ListenAndServe(":9090", nil)
	if err != nil {
		log.Fatal("聊天服务启动失败！！！:", err)
	}
}
