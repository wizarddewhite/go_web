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
	this.Data["Title"] = "account"
	this.Data["IsLogin"], this.Data["IsAdmin"] = checkAccount(this.Ctx)

	if isReg {
		this.TplName = "account_reg.html"
	}

	// only login user could view his account
	if !this.Data["IsLogin"].(bool) {
		this.Ctx.SetCookie("flash", "Please Login first", 1024, "/")
		this.Redirect("/login", 301)
		return
	}

	if this.Data["IsAdmin"].(bool) {
		this.Data["Users"], _ = models.GetAllUsers()
		this.TplName = "account.html"
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
