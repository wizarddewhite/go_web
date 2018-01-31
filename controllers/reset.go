package controllers

import (
	"go_web/models"

	"github.com/astaxie/beego"
	"golang.org/x/crypto/bcrypt"
	//"github.com/astaxie/beego/context"
)

type ResetController struct {
	beego.Controller
}

func (this *ResetController) Get() {
	uname := this.Input().Get("uname")
	hash := this.Input().Get("hash")

	this.Data["Title"] = "Reset Password"
	this.Data["User"] = uname
	this.Data["Hash"] = hash
	this.TplName = "reset_pass.html"
	return
}

func (this *ResetController) Post() {
	uname := this.Input().Get("uname")
	hash := this.Input().Get("hash")
	pwd := this.Input().Get("pwd")

	ph, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.DefaultCost)
	if err != nil {
		this.Ctx.SetCookie("flash", "Password is not valid", 1024, "/")
		this.Redirect("/", 301)
		return
	}
	err = models.ResetUser(uname, hash, string(ph))
	if err != nil {
		this.Ctx.SetCookie("flash", err.Error(), 1024, "/")
	} else {
		this.Ctx.SetCookie("flash", "Password reset done", 1024, "/")
	}
	this.Redirect("/login", 301)
	return
}
