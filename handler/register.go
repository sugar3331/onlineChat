package im

import (
	"OnlineChat/tools"
	"context"
	"encoding/json"
	"fmt"

	"html/template"
	"net/http"
)

type register struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

func Register(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		tmpl := template.Must(template.ParseFiles("./static/register.html"))
		tmpl.Execute(w, nil)
	} else if r.Method == "POST" {
		var user register
		err := json.NewDecoder(r.Body).Decode(&user)
		if err != nil {
			// 处理错误
			http.Error(w, "Failed to parse JSON data", http.StatusBadRequest)
			return
		}
		resp := &HttpCode{}
		ret := Registered(&user)
		if ret == 1 {
			resp = &HttpCode{
				Code:    10002,
				Message: "注册失败,用户已存在",
				Data:    struct{}{},
			}
		} else if ret == 2 {
			resp = &HttpCode{
				Code:    10003,
				Message: "注册失败,请重新注册",
				Data:    struct{}{},
			}
		} else {
			resp = &HttpCode{
				Code:    0,
				Message: "注册成功,请前往登录",
				Data:    struct{}{},
			}
		}
		jsonData, err := json.Marshal(resp)
		if err != nil {
			http.Error(w, "Failed to convert to JSON", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonData)
		return
	}
}

func Registered(req *register) int {
	user := &User{}
	sql := "SELECT Id,UserName,Name FROM user WHERE UserName = ?"
	err := GlobalConn.QueryRowCtx(context.Background(), user, sql, req.Username)
	if err == nil {
		return 1
	}

	//var worker *Worker
	worker := tools.NewWorker(001, 002)
	//ID:=gg.NextID()
	newId, _ := worker.NextID() // 使用雪花算法生成新的Id
	fmt.Println("newId:")
	fmt.Println(newId)

	sql = "INSERT INTO user (`Id`, `UserName`, `PassWord`, `Name`) VALUES (?, ?, ?,?)"
	r, err := GlobalConn.ExecCtx(context.Background(), sql, newId, req.Username, req.Password, req.Name)
	if err != nil {
		panic(err)
		return 2
	}
	fmt.Println(r.RowsAffected())
	return 0
}
