package controllers

import (
	"bytes"
	"net/smtp"
	"regexp"
	"strconv"
	"strings"
	"text/template"

	"go_web/models"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/utils/pagination"
	"golang.org/x/crypto/bcrypt"
	//"github.com/astaxie/beego/context"
)

type AccountController struct {
	beego.Controller
}

func (this *AccountController) Get() {
	isReg := this.Input().Get("reg") == "true"
	this.Data["Title"] = "account"
	// get account information
	user := getLoginUser(&this.Controller)
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

	// only login user could view his account
	if !this.Data["IsLogin"].(bool) {
		this.Ctx.SetCookie("flash", "Please Login first", 1024, "/")
		this.Redirect("/login", 301)
		return
	}

	if this.Data["IsAdmin"].(bool) {
		var total_users int64
		curr_page := 1
		p := this.Input().Get("p")
		p_num, err := strconv.ParseInt(p, 10, 64)
		if err == nil {
			curr_page = int(p_num)
		}
		usersPerPage := 13

		// admin could view all users' informatioin
		this.Data["Users"], total_users, _ = models.GetAllUsers(usersPerPage,
			int64(usersPerPage*(curr_page-1)))

		// sets this.Data["paginator"]
		paginator := pagination.SetPaginator(this.Ctx, usersPerPage,
			total_users)
		this.Data["paginator"] = paginator
		this.TplName = "account.html"
	} else {
		// normal user only view his informatioin
		this.Data["User"] = user
		this.TplName = "account.html"
	}
}

func validateEmail(email string) bool {
	Re := regexp.MustCompile(`^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,4}$`)
	return Re.MatchString(email)
}

func (this *AccountController) ConfirmEmail() {
	uname := this.Input().Get("uname")
	hash := this.Input().Get("hash")

	err := models.VerifyUserEmail(uname, hash)
	if err == nil {
		this.Ctx.SetCookie("flash", "Email Confirmed!", 1024, "/")
	} else {
		this.Ctx.SetCookie("flash", err.Error(), 1024, "/")
	}
	this.Redirect("/login", 301)
}

func (this *AccountController) Post() {
	uname := this.Input().Get("uname")
	email := strings.ToLower(this.Input().Get("email"))
	pwd := this.Input().Get("pwd")
	hash, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.DefaultCost)
	uid := this.Input().Get("uid")
	if len(uid) == 0 {
		// create a new account
		if err != nil {
			this.Redirect("/account?reg=true", 301)
			return
		}

		// check uname, pwd, email
		if len(uname) == 0 || len(pwd) == 0 || len(email) == 0 {
			this.Ctx.SetCookie("flash", "Check your name, password or email", 1024, "/")
			this.Redirect("/account?reg=true", 301)
			return
		}

		if !validateEmail(email) {
			this.Ctx.SetCookie("flash", "Check your email", 1024, "/")
			this.Redirect("/account?reg=true", 301)
			return
		}

		err, vh, _ := models.AddUser(uname, email, string(hash))
		if err != nil {
			beego.Trace(err)
			this.Ctx.SetCookie("flash", err.Error(), 1024, "/")
			this.Redirect("/account?reg=true", 301)
		} else {
			this.Ctx.SetCookie("flash", "A confirmation mail sent to your box, please confirm", 1024, "/")
			this.Redirect("/login", 301)
			RequestConfirm(uname, email, vh)
		}
	} else {
		// modify an account
		if err != nil {
			this.Redirect("/account/edit/"+uid, 301)
			return
		}

		// change password
		if len(pwd) != 0 {
			err = models.ModifyUserSec(uname, string(hash))
			if err != nil {
				this.Redirect("/account/edit/"+uid, 301)
			}
		}

		this.Redirect("/account", 301)
	}

	return
}

func (this *AccountController) Edit() {
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
}

func (this *AccountController) ResetBD() {
	// only admin could invoke this
	user := getLoginUser(&this.Controller)
	if user == nil {
		this.Ctx.SetCookie("flash", "Please Login first", 1024, "/")
		this.Redirect("/login", 301)
		return
	} else if !user.IsAdmin {
		this.Ctx.SetCookie("flash", "No page found", 1024, "/")
		this.Redirect("/login", 301)
		return
	}

	uid, _ := strconv.ParseInt(this.Ctx.Input.Params()["0"], 10, 64)
	if uid != user.Id {
		user = models.GetUserById(uid)
	}
	// Reset user bandwidth usage
	models.ModifyUserStat(user.Name, "-1", "-1")
	this.Ctx.SetCookie("flash", user.Name+" BD Flashed", 1024, "/")
	this.Redirect("/account", 301)
}

func (this *AccountController) ExpandOM() {
	// only admin could invoke this
	user := getLoginUser(&this.Controller)
	if user == nil {
		this.Ctx.SetCookie("flash", "Please Login first", 1024, "/")
		this.Redirect("/login", 301)
		return
	} else if !user.IsAdmin {
		this.Ctx.SetCookie("flash", "No page found", 1024, "/")
		this.Redirect("/login", 301)
		return
	}

	uid, _ := strconv.ParseInt(this.Ctx.Input.Params()["0"], 10, 64)
	if uid != user.Id {
		user = models.GetUserById(uid)
	}
	// Reset user bandwidth usage
	models.ExpandUserExpire(user.Name, 1)
	this.Ctx.SetCookie("flash", user.Name+" Expire Flashed", 1024, "/")
	this.Redirect("/account", 301)
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
		this.Ctx.SetCookie("flash", "User not deleted", 1024, "/")
		beego.Error(err)
	} else {
		this.Ctx.SetCookie("flash", "User deleted", 1024, "/")
	}
DONE:
	this.Redirect("/account", 302)
}

/**
 * user : example@example.com login smtp server user
 * password : xxxxx login smtp server password
 * host : smtp.example.com:port smtp.163.com:25
 * to : example@example.com;example1@163.com;example2@sina.com.cn;...
 * subject : The subject of mail
 * body : The content of mail
 * mailtyoe : mail type html or text
**/
func send(to, subject, body, mailtype string) error {
	user := beego.AppConfig.String("email")
	password := beego.AppConfig.String("password")
	host := beego.AppConfig.String("smtp")

	hp := strings.Split(host, ":")
	auth := smtp.PlainAuth("", user, password, hp[0])
	var content_type string
	if mailtype == "html" {
		content_type = "Content-Type: text/" + mailtype + "; charset=UTF-8"
	} else {
		content_type = "Content-Type: text/plain" + "; charset=UTF-8"
	}

	msg := "From: Webframe <no-reply@mail.com> \n" +
		"To: " + to + "\n" +
		content_type + "\n" +
		"Subject: " + subject + "\n\n" +
		body
	send_to := strings.Split(to, ";")
	err := smtp.SendMail(host, auth, user, send_to, []byte(msg))
	return err
}

type ConfimrMail struct {
	Name string
	Hash string
}

func RequestConfirm(uname, to, hash string) {

	Templ := `
<html>
<body>
    <h3>Email confirmation:</h3>
    <p>Someone has register an account on ShebaoGongjijin.tk, confirm it or ignore.</p>
    <a href="http://shebaogongjijin.tk/account/confirmemail?uname={{.Name}}&hash={{.Hash}}">Click to Confirm</a>
</body>
</html>
`

	var body bytes.Buffer
	t, _ := template.New("cm").Parse(Templ)
	t.Execute(&body, &ConfimrMail{uname, hash})
	err := send(to, "ShebaoGongjijin: Account Confirmation", body.String(), "html")
	if err != nil {
		beego.Error(err)
	}
}
