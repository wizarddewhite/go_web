package controllers

import (
	"bufio"
	"go_web/models"
	"os"
	"os/exec"
	"strconv"
	"strings"

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

		// create a linux account
		useradd := "useradd -s /bin/true -d /home/" + uname + " -m " + uname
		cmd := exec.Command("bash", "-c", useradd)
		_, err := cmd.Output()
		if err != nil {
			s := strings.Split(err.Error(), " ")
			if len(s) == 3 && s[2] == "9" {
				this.Ctx.SetCookie("flash", "User already exist, change another name", 1024, "/")
			} else {
				this.Ctx.SetCookie("flash", "System Error, try again", 1024, "/")
			}
			this.Redirect("/account?reg=true", 301)
			return
		}

		// mkdir
		cmd = exec.Command("bash", "-c", "mkdir -p /home/"+uname+"/.ssh")
		cmd.Output()
		// add to group
		cmd = exec.Command("bash", "-c", "usermod -g "+uname+" -G ssh "+uname)
		cmd.Output()
		// touch authorized_keys
		cmd = exec.Command("bash", "-c", "touch /home/"+uname+"/.ssh/authorized_keys")
		cmd.Output()
		// chown
		cmd = exec.Command("bash", "-c", "chown -R "+uname+":"+uname+" /home/"+uname+"/.ssh")
		cmd.Output()

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

		// change password
		if len(pwd) != 0 {
			err = models.ModifyUserSec(uname, string(hash))
			if err != nil {
				this.Redirect("/account/modify/"+uid, 301)
			}
		}

		// add a new key
		new_key := this.Input().Get("key")
		user := models.GetUser(uname)
		if len(new_key) != 0 && user != nil && user.NumKeys < user.KeyLimit {
			file, err := os.OpenFile("/home/"+uname+"/.ssh/authorized_keys", os.O_APPEND|os.O_WRONLY, 0600)
			if err != nil {
				this.Redirect("/account/modify/"+uid, 301)
				return
			}

			defer file.Close()

			if _, err = file.WriteString(new_key + "\n"); err != nil {
				this.Redirect("/account/modify/"+uid, 301)
				return
			}
			// update the db
			models.ModifyUserKey(uname, 1)
		}
		this.Redirect("/account", 301)
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
	} else {
		// delete user
		cmd := exec.Command("bash", "-c", "userdel "+user.Name)
		cmd.Output()
		// remote dir
		cmd = exec.Command("bash", "-c", "rm -rf /home/"+user.Name)
		cmd.Output()
	}
	this.Ctx.SetCookie("flash", "User deleted", 1024, "/")
DONE:
	this.Redirect("/account", 302)
}

func (this *AccountController) DeleteKey() {
	var file *os.File
	var err error
	var scanner *bufio.Scanner
	var keys []string
	var index int64

	user := getLoginUser(&this.Controller)
	if user == nil {
		this.Ctx.SetCookie("flash", "Please Login first", 1024, "/")
		this.Redirect("/login", 301)
		return
	}

	if len(this.Ctx.Input.Params()["0"]) == 0 {
		goto DONE
	}

	index, _ = strconv.ParseInt(this.Ctx.Input.Params()["0"], 10, 64)

	// read key again
	file, err = os.OpenFile("/home/"+user.Name+"/.ssh/authorized_keys", os.O_RDWR, 0600)
	if err != nil {
		goto DONE
	}
	defer file.Close()
	scanner = bufio.NewScanner(file)
	for scanner.Scan() {
		keys = append(keys, scanner.Text())
	}

	// if the index is out of range, return
	if index >= int64(len(keys)) {
		goto DONE
	}

	// truncate file to 0 and write keys back except the skipped one
	err = file.Truncate(0)
	if err != nil {
		goto DONE
	}
	file.Seek(0, 0)

	for i, key := range keys {
		if int64(i) != index {
			file.WriteString(key + "\n")
		}
	}

DONE:
	this.Redirect("/account", 302)
	return
}
