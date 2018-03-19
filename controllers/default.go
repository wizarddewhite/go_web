package controllers

import (
	"github.com/astaxie/beego"
)

type MainController struct {
	beego.Controller
}

func (this *MainController) Get() {
	this.TplName = "home.html"
	this.Data["Title"] = "Shebao"
	this.Data["IsHome"] = true
	// get account information
	getLoginUser(&this.Controller)
	ck, err := this.Ctx.Request.Cookie("flash")
	if err == nil {
		this.Data["Flash"] = ck.Value
		this.Ctx.SetCookie("flash", "", -1, "/")
	}
	beego.Trace("home/get")
}
