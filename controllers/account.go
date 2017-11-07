package controllers

import (
	"go_web/models"

	"github.com/astaxie/beego"
	"golang.org/x/crypto/bcrypt"
	//"github.com/astaxie/beego/context"
)

type AccountController struct {
	beego.Controller
}

func (this *AccountController) Get() {
	isReg := this.Input().Get("reg") == "true"

	if isReg {
		this.TplName = "account_reg.html"
	} else {
		ck, err := this.Ctx.Request.Cookie("uname")
		if err == nil {
			uname := ck.Value
			user := models.GetUser(uname)
			this.Data["User"] = user
		} else {
			this.Data["User"] = nil
		}
		this.TplName = "account.html"
	}
	this.Data["Title"] = "account"
	this.Data["IsLogin"] = checkAccount(this.Ctx)
}

func (this *AccountController) Post() {
	uname := this.Input().Get("uname")
	pwd := this.Input().Get("pwd")
	hash, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.DefaultCost)
	if err != nil {
		this.Redirect("/account?reg=true", 301)
		return
	}

	err = models.AddUser(uname, string(hash))
	if err != nil {
		this.Redirect("/account?reg=true", 301)
	} else {
		this.Redirect("/login", 301)
	}

	return
}
