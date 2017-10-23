package controllers

import (
	"go_web/models"

	"github.com/astaxie/beego"
	"golang.org/x/crypto/bcrypt"
)

type LoginController struct {
	beego.Controller
}

func (this *LoginController) Get() {
	isExit := this.Input().Get("exit") == "true"

	if isExit {
		beego.Trace("login/isexit")
		this.Ctx.SetCookie("uname", "", -1, "/")
		this.Ctx.SetCookie("pwd", "", -1, "/")
		this.Redirect("/", 301)
		return
	}

	user := getLoginUser(&this.Controller)
	if user != nil {
		this.Ctx.SetCookie("flash", "Already Login", 1024, "/")
		this.Redirect("/", 301)
		return
	}

	this.TplName = "login.html"
	this.Data["Title"] = "login"
	ck, err := this.Ctx.Request.Cookie("flash")
	if err == nil {
		this.Data["Flash"] = ck.Value
		this.Ctx.SetCookie("flash", "", -1, "/")
	}
}

func pwd_same(pwd_hash, pwd string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(pwd_hash), []byte(pwd))
	if err != nil {
		return false
	} else {
		return true
	}
}

func (this *LoginController) Post() {
	//this.Ctx.WriteString(fmt.Sprint(this.Input()))
	uname := this.Input().Get("uname")
	pwd := this.Input().Get("pwd")
	autoLogin := this.Input().Get("autoLogin") == "on"

	user := models.GetUser(uname)
	if user != nil && user.VHash != "v" {
		this.Ctx.SetCookie("flash", "Please verify your account first", 1024, "/")
		this.Redirect("/", 301)
	} else if user != nil && user.Name == uname && pwd_same(user.PWD, pwd) && user.VHash == "v" {
		maxAge := 0
		if autoLogin {
			maxAge = 1<<31 - 1
		}

		this.Ctx.SetCookie("uname", uname, maxAge, "/")
		this.Ctx.SetCookie("pwd", pwd, maxAge, "/")
		this.Redirect("/account", 301)
	} else {
		this.Ctx.SetCookie("flash", "Username or Password error", 1024, "/")
		this.Redirect("/", 301)
	}
	return
}

func getLoginUser(this *beego.Controller) *models.User {
	var uname, pwd string
	var user *models.User
	ck, err := this.Ctx.Request.Cookie("uname")
	if err != nil {
		goto NOLOGIN
	}
	uname = ck.Value
	ck, err = this.Ctx.Request.Cookie("pwd")
	if err != nil {
		goto NOLOGIN
	}
	pwd = ck.Value

	user = models.GetUser(uname)
	if user != nil && pwd_same(user.PWD, pwd) {
		this.Data["IsLogin"] = true
		this.Data["IsAdmin"] = user.IsAdmin
		return user
	}

NOLOGIN:
	this.Data["IsLogin"] = false
	this.Data["IsAdmin"] = false
	return nil
}
