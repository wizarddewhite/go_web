package controllers

import (
	"github.com/astaxie/beego"

	"bihu_helper/models"
)

type StatusController struct {
	beego.Controller
}

func (this *StatusController) Get() {
	this.TplName = "status.html"
	this.Data["Title"] = "WebFrame"
	// get account information
	getLoginUser(&this.Controller)
	ck, err := this.Ctx.Request.Cookie("flash")
	if err == nil {
		this.Data["Flash"] = ck.Value
		this.Ctx.SetCookie("flash", "", -1, "/")
	}
	this.Data["Posts"], _, _ = models.GetAllPosts(10, 0)
}
