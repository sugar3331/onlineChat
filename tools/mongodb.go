package tools

import (
	"context"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
)

var GolbalMogodb *mongo.Client

func MongodbInit() {
	go func() {
		client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI("mongodb://root:hawk123@1.94.27.198:27017"))
		err = client.Ping(context.TODO(), nil)
		if err != nil {
			// 处理连接失败的错误
			log.Println("mongodb 连接数据库失败:", err)
		} else {
			log.Println("mongodb 连接数据库成功")
		}
		GolbalMogodb = client
	}()
}
