package im

import (
	"OnlineChat/tools"
	"context"
	"encoding/json"
	"fmt"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"html/template"
	"net/http"
	"strconv"
)

const (
	OK          = 0
	NotLogin    = 10001 //您还没有登录
	UserInfoErr = 10002 //用户信息错误
	DoErr       = 10003

	NotFound = 10004 //信息不存在
)

var GlobalConn sqlx.SqlConn

func NewMysqlInit() {
	go func() {
		dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8&parseTime=True&loc=Local", "root", "hawk123", "1.94.27.198:3306", "hawk")
		conn := sqlx.NewMysql(dsn)
		GlobalConn = conn
	}()
}

type login struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type Token struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func Login(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		// 渲染登录页面
		tmpl := template.Must(template.ParseFiles("./static/login.html"))
		tmpl.Execute(w, nil)
	} else if r.Method == "POST" {
		var user login
		err := json.NewDecoder(r.Body).Decode(&user)
		if err != nil {
			// 处理错误
			http.Error(w, "Failed to parse JSON data", http.StatusBadRequest)
			return
		}
		dbRet, code := getUser(user.Username, user.Password)
		fmt.Println(dbRet)
		if code == 1 {

			resp := &HttpCode{
				Code:    UserInfoErr,
				Message: "登录失败",
				Data:    struct{}{},
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
		idInt64, _ := strconv.ParseInt(dbRet.Id, 10, 64)
		a, err := tools.Token.GetToken(idInt64, dbRet.UserName, dbRet.Name, "user")
		if err != nil {
			resp := &HttpCode{
				Code:    UserInfoErr,
				Message: "登录失败",
				Data:    struct{}{},
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
		resp := &HttpCode{
			Code:    OK,
			Message: "登录成功",
			Data: Token{
				AccessToken:  a,
				RefreshToken: "",
			},
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

type User struct {
	Id       string `json:"id",db:"Id"`
	UserName string `json:"userName",db:"UserName"`
	Name     string `json:"name",db:"Name"`
}

type HttpCode struct {
	Message string      `json:"message"`
	Code    int         `json:"code"`
	Data    interface{} `json:"data"`
}

func getUser(name, pwd string) (*User, int) {
	user := &User{}
	sql := "SELECT `Id`, `Name`, `UserName` FROM `user` WHERE `UserName` = ? AND `password` = ? LIMIT 1"
	err := GlobalConn.QueryRowCtx(context.Background(), user, sql, name, pwd)
	if err != nil {
		fmt.Printf("err:%s\n", err)

		return nil, 1
	}
	return user, 0
}
