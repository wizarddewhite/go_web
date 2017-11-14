package controllers

import (
	"bufio"
	"go_web/models"
	"os"
	"strconv"

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
	ck, err := this.Ctx.Request.Cookie("flash")
	if err == nil {
		this.Data["Flash"] = ck.Value
		this.Ctx.SetCookie("flash", "", -1, "/")
	}

	// register a new user
	if isReg {
		this.TplName = "account_reg.html"
		return
	}

	// get account information
	user := getLoginUser(&this.Controller)
	// only login user could view his account
	if !this.Data["IsLogin"].(bool) {
		this.Ctx.SetCookie("flash", "Please Login first", 1024, "/")
		this.Redirect("/login", 301)
		return
	}

	if this.Data["IsAdmin"].(bool) {
		// admin could view all users' informatioin
		this.Data["Users"], _ = models.GetAllUsers()
		this.TplName = "account.html"
	} else {
		// normal user only view his informatioin
		this.Data["User"] = user
		this.TplName = "account.html"
	}
}

func (this *AccountController) Post() {
	uname := this.Input().Get("uname")
	pwd := this.Input().Get("pwd")
	hash, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.DefaultCost)
	uid := this.Input().Get("uid")
	if len(uid) == 0 {
		// create a new account
		if err != nil {
			this.Redirect("/account?reg=true", 301)
			return
		}

		_, isAdmin := checkAccount(this.Ctx)
		if !isAdmin {
			this.Ctx.SetCookie("flash", "Contact Admin to register", 1024, "/")
			this.Redirect("/", 301)
			return
		}

		err = models.AddUser(uname, string(hash))
		if err != nil {
			this.Redirect("/account?reg=true", 301)
		} else {
			this.Redirect("/login", 301)
		}
	} else {
		// modify an account
		if err != nil {
			this.Redirect("/account/modify/"+uid, 301)
			return
		}

		err = models.ModifyUserSec(uname, string(hash), "holder")
		if err != nil {
			this.Redirect("/account/modify/"+uid, 301)
		} else {
			this.Redirect("/account", 301)
		}
	}

	return
}

func (this *AccountController) Modify() {
	user := getLoginUser(&this.Controller)
	if user == nil {
		this.Ctx.SetCookie("flash", "Please Login first", 1024, "/")
		this.Redirect("/login", 301)
		return
	}

	// could only edit your account for normal user
	uid, _ := strconv.ParseInt(this.Ctx.Input.Params()["0"], 10, 64)
	if uid != user.Id && !user.IsAdmin {
		this.Ctx.SetCookie("flash", "Operation not permitted", 1024, "/")
		this.Redirect("/", 301)
		return
	}

	if user.IsAdmin && uid != user.Id {
		user = models.GetUserById(uid)
	}
	this.TplName = "account_modify.html"
	this.Data["Title"] = "Modify account"
	this.Data["Account"] = user
	this.Data["Uid"] = user.Id

	// retrieve keys
	file, err := os.Open("/home/" + user.Name + "/.ssh/authorized_keys")
	if err != nil {
		return
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	var keys []string
	for scanner.Scan() {
		keys = append(keys, scanner.Text())
	}
	this.Data["Keys"] = keys
}

func (this *AccountController) Delete() {
	var user *models.User
	var uid int64
	var err error
	if len(this.Ctx.Input.Params()["0"]) == 0 {
		this.Ctx.SetCookie("flash", "Page not found", 1024, "/")
		goto DONE
	}

	// only admin could delete user
	user = getLoginUser(&this.Controller)
	if user == nil || !user.IsAdmin {
		this.Ctx.SetCookie("flash", "Page not found", 1024, "/")
		goto DONE
	}

	uid, _ = strconv.ParseInt(this.Ctx.Input.Params()["0"], 10, 64)
	user = models.GetUserById(uid)
	if user == nil {
		this.Ctx.SetCookie("flash", "No such User", 1024, "/")
		goto DONE
	}

	err = models.DeleteUser(this.Ctx.Input.Params()["0"])
	if err != nil {
		beego.Error(err)
	}
	this.Ctx.SetCookie("flash", "User deleted", 1024, "/")
DONE:
	this.Redirect("/account", 302)
}
