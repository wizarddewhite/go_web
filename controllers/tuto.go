package controllers

import (
	"github.com/astaxie/beego"
)

type TutoController struct {
	beego.Controller
}

func (this *TutoController) Get() {
	this.TplName = "home.html"
	this.Data["Title"] = "Tutorial"
	this.Data["IsTuto"] = true
	this.Data["IsLogin"], this.Data["IsAdmin"] = checkAccount(this.Ctx)
	ck, err := this.Ctx.Request.Cookie("flash")
	if err == nil {
		this.Data["Flash"] = ck.Value
		this.Ctx.SetCookie("flash", "", -1, "/")
	}
}
