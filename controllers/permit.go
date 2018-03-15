package controllers

import (
	"go_web/models"

	"github.com/astaxie/beego"
)

type PermitController struct {
	beego.Controller
}

type Perm struct {
	NumEX   int
	NumCoin int
}

func (this *PermitController) Get() {
	var perm Perm
	uname := this.Input().Get("uname")
	uuid := this.Input().Get("uuid")
	u := models.GetUser(uname)

	if u != nil && u.UUID == uuid {
		switch level := u.Level; level {
		case "beginner":
			perm = Perm{3, 5}
		case "standard":
			perm = Perm{5, 10}
		case "advanced":
			perm = Perm{-1, -1}
		default:
			perm = Perm{2, 1}
		}
	} else {
		perm = Perm{2, 1}
	}

	this.Data["json"] = perm
	this.ServeJSON()
}
