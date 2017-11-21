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
	ip := nodes.GetNode()
	this.Ctx.WriteString(uname + "@" + ip)
}
