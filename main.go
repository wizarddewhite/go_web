package main

import (
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/logs"
	"github.com/astaxie/beego/orm"
	"go_web/models"
	"go_web/nodes"
	_ "go_web/routers"
)

func init() {
	models.RegisterDB()
}

func main() {
	if len(beego.AppConfig.String("key")) == 0 {
		beego.Error("need key")
		return
	}
	orm.Debug = true
	orm.RunSyncdb("default", false, true)

	logs.SetLogger(logs.AdapterFile, `{"filename":"logs/freeland.log","level":7,"maxlines":0,"maxsize":0,"daily":true,"maxdays":10}`)

	// setup master
	err := nodes.GetMaster()
	if err != nil {
		beego.Trace("error on setup master")
		return
	}

	// read api to check already booted server
	err = nodes.RetrieveNodes()
	if err != nil {
		return
	}

	beego.Run()
}
