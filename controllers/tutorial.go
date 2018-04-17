package controllers

import (
	"github.com/astaxie/beego"
)

type TutorialController struct {
	beego.Controller
}

func (this *TutorialController) Get() {
	this.TplName = "tutorial.html"
}
