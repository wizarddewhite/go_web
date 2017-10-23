package main

import (
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/logs"
	"github.com/astaxie/beego/orm"
	"github.com/robfig/cron"

	"go_web/models"
	_ "go_web/routers"
)

func init() {
	models.RegisterDB()
}

func user_refill() {
	ru, err := models.RefillUsers()
	if err == nil {
		for _, u := range ru {
			models.RefillUser(u)
		}
	}
}

func main() {
	orm.Debug = true
	orm.RunSyncdb("default", false, true)

	logs.SetLogger(logs.AdapterFile, `{"filename":"logs/freeland.log","level":7,"maxlines":0,"maxsize":0,"daily":true,"maxdays":10}`)

	// User Refill task
	c := cron.New()
	spec := "0 */10 * * * *"
	c.AddFunc(spec, user_refill)
	c.Start()

	beego.Run()
}
