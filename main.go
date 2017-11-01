package main

import (
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/orm"
	"web/models"
	_ "web/routers"
)

func init() {
	models.RegisterDB()
}

func main() {
	orm.Debug = true
	orm.RunSyncdb("default", false, true)
	beego.Run()
}
