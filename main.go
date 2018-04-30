package main

import (
	"time"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/logs"
	"github.com/astaxie/beego/orm"
	//"github.com/robfig/cron"

	"bihu_helper/models"
	_ "bihu_helper/routers"
)

func init() {
	models.RegisterDB()
}

var pop_star = []string{
	"179159", // me
	"9909",   // jinma
	"1385",   // 爱思考的糖
	"131507", // 圊呓语
	"483",    // 玩火的猴子
	"2234",   // 南宫远
	"11880",  // 湘乡的大树
	"55332",  // 吴庆英
	"12627",  // jimi
	"193646", // wdctll
	"9457",   // Bean
	"13599",  // 陈竹
	"41279",  // 串串
}

func main() {
	orm.Debug = true
	orm.RunSyncdb("default", false, true)

	logs.SetLogger(logs.AdapterFile, `{"filename":"logs/freeland.log","level":7,"maxlines":0,"maxsize":0,"daily":true,"maxdays":10}`)

	go models.Update_Proxy()
	models.BH_retrieve_ip()

	models.QF = make(chan models.QueryFollow, 10)
	models.QU = make(chan int, 10)
	time.Sleep(5 * time.Second)
	time.Sleep(time.Duration(models.Raw_Proxys*3/100) * time.Second)
	// models.BH_update_db()
	go models.Upvote_BH(models.QU)
	go models.BH_up_vote()

	beego.Run()
}
