package main

import (
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/orm"
	"go_web/models"
	_ "go_web/routers"
)

func init() {
	models.RegisterDB()
}

func main() {
	orm.Debug = true
	orm.RunSyncdb("default", false, true)
	beego.Run()
}
