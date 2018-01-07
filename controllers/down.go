package controllers

import (
	"github.com/astaxie/beego"
)

type DownController struct {
	beego.Controller
}

func (this *DownController) Get() {
	this.TplName = "home.html"
	this.Data["Title"] = "Download"
	this.Data["IsDown"] = true
	this.Data["IsLogin"], this.Data["IsAdmin"] = checkAccount(this.Ctx)
	ck, err := this.Ctx.Request.Cookie("flash")
	if err == nil {
		this.Data["Flash"] = ck.Value
		this.Ctx.SetCookie("flash", "", -1, "/")
	}
}
