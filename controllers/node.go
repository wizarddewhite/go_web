package controllers

import (
	"github.com/astaxie/beego"
	"go_web/nodes"
)

type NodeController struct {
	beego.Controller
}

func (this *NodeController) Get() {
	uname := this.Input().Get("uname")
	ip := nodes.GetServiceNode()
	this.Ctx.WriteString(uname + "@" + ip + "\n")
}

func (this *NodeController) Renew() {
	if this.Ctx.Input.IP() == "127.0.0.1" {
		this.Data["json"] = "{\"Status\":\"ok\"}"
		nodes.AddTask("", "", "renew_cert")
		nodes.AccSync()
	} else {
		this.Data["json"] = "{\"Status\":\"err\"}"
	}
	this.ServeJSON()
	return
}
