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
	this.Data["IsLogin"] = checkAccount(this.Ctx)
	topics, err := models.GetAllTopics(true)
	if err != nil {
		beego.Error(err)
	} else {
		this.Data["Topics"] = topics
	}
	beego.Trace("home/get")
}
