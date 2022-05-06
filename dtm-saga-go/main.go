package main

import (
	"errors"
	"fmt"
	"github.com/dtm-labs/dtmcli"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"log"

	"time"
)

// 事务参与者的服务地址
const BusiAPI = "/api/busi_start"
const BusiPort = 8000

var Busi = fmt.Sprintf("http://localhost:%d%s", BusiPort, BusiAPI)

func main() {
	StartServe()
	_ = QsFireRequest()
	select {}
}

// StartServe quick start: start server
func StartServe() {
	app := gin.New()
	qsAddRoute(app)
	log.Printf("quick start examples listening at %d", BusiPort)
	go func() {
		_ = app.Run(fmt.Sprintf(":%d", BusiPort))
	}()
	time.Sleep(100 * time.Millisecond)
}

func qsAddRoute(app *gin.Engine) {
	bank1Repo := Repo{NewMysqlDB("bank1")}
	bank2Repo := Repo{NewMysqlDB("bank2")}

	app.POST(BusiAPI+"/minus-zs-balances", func(c *gin.Context) {
		err := bank1Repo.UpdateBalances(AccountEvent{
			AccountNo: 1,  // 1:zs
			Amount:    -1, // -1 代表扣减金额
		})
		if err != nil {
			c.JSON(500, err.Error())
		} else {
			c.JSON(200, "")
		}
	})

	app.POST(BusiAPI+"/add-zs-balances", func(c *gin.Context) {
		err := bank1Repo.UpdateBalances(AccountEvent{
			AccountNo: 1, // 1:zs
			Amount:    1,
		})
		if err != nil {
			c.JSON(500, err.Error())
		} else {
			c.JSON(200, "")
		}
	})

	app.POST(BusiAPI+"/add-ls-balances", func(c *gin.Context) {
		if true {
			c.JSON(409, errors.New("manually")) // Status 409 表示失败，不再重试，直接回滚，这个是框架定的
			return
		}

		err := bank2Repo.UpdateBalances(AccountEvent{
			AccountNo: 2, // 2:lisi
			Amount:    1,
		})
		if err != nil {
			c.JSON(500, err.Error())
		} else {
			c.JSON(200, "")
		}
	})

	app.POST(BusiAPI+"/minus-ls-balances", func(c *gin.Context) {
		err := bank2Repo.UpdateBalances(AccountEvent{
			AccountNo: 2, // 2:lisi
			Amount:    -1,
		})
		if err != nil {
			c.JSON(500, err.Error())
		} else {
			c.JSON(200, "")
		}
	})
}

const dtmServer = "http://localhost:36789/api/dtmsvr"

// QsFireRequest quick start: fire request
func QsFireRequest() string {
	req := &gin.H{"amount": 30} // 微服务的载荷
	// DtmServer为DTM服务的地址
	saga := dtmcli.NewSaga(dtmServer, dtmcli.MustGenGid(dtmServer)).
		// 添加一个TransOut的子事务，正向操作为url: Busi+"/minus-zs-balances"， 逆向操作为url: Busi+"/add-zs-balances"
		Add(Busi+"/minus-zs-balances", Busi+"/add-zs-balances", req).
		// 添加一个TransIn的子事务，正向操作为url: Busi+"/add-ls-balances"， 逆向操作为url: Busi+"/minus-ls-balances"
		Add(Busi+"/add-ls-balances", Busi+"/minus-ls-balances", req)
	// 提交saga事务，dtm会完成所有的子事务/回滚所有的子事务
	err := saga.Submit()

	if err != nil {
		panic(err)
	}
	log.Printf("transaction: %s submitted", saga.Gid)
	return saga.Gid
}

func NewMysqlDB(database string) *gorm.DB {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local",
		"root",
		"123456",
		"localhost",
		3306,
		database,
		"utf8mb4")

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("start mysql client err: %s" + err.Error())
	}
	return db
}