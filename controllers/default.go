package controllers

import (
	"go_web/models"

	"github.com/astaxie/beego"
)

type MainController struct {
	beego.Controller
}

func (this *MainController) Get() {
	this.TplName = "home.html"
	this.Data["Title"] = "home"
	this.Data["IsHome"] = true
	this.Data["IsLogin"], this.Data["IsAdmin"] = checkAccount(this.Ctx)
	ck, err := this.Ctx.Request.Cookie("flash")
	if err == nil {
		this.Data["Flash"] = ck.Value
		this.Ctx.SetCookie("flash", "", -1, "/")
	}
	topics, err := models.GetAllTopics(this.Input().Get("cate"), true)
	if err != nil {
		beego.Error(err)
	} else {
		this.Data["Topics"] = topics
	}
	categories, err := models.GetAllCategories()
	if err != nil {
		beego.Error(err)
	} else {
		this.Data["Categories"] = categories
	}
	beego.Trace("home/get")
}
