package controllers

import (
	"github.com/astaxie/beego"
)

type MainController struct {
	beego.Controller
}

func (c *MainController) Get() {
	c.TplName = "home.html"
	c.Data["Title"] = "home"
	c.Data["IsHome"] = true
	beego.Trace("home/get")
}
