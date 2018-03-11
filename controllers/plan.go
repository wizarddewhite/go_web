package controllers

import (
	"github.com/astaxie/beego"
)

type PlanController struct {
	beego.Controller
}

func (this *PlanController) Get() {
	this.TplName = "plan.html"
	this.Data["Title"] = "Plan"
	this.Data["IsPlan"] = true
}
