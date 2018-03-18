package controllers

import (
	"bufio"
	"bytes"
	"go_web/models"
	"go_web/nodes"
	"net/smtp"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"text/template"

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

		// password
		cmd = exec.Command("bash", "-c", "usermod -p '*' "+uname)
		cmd.Output()
		// mkdir
		cmd = exec.Command("bash", "-c", "mkdir -p /home/"+uname+"/.ssh")
		cmd.Output()
		// touch authorized_keys
		cmd = exec.Command("bash", "-c", "touch /home/"+uname+"/.ssh/authorized_keys")
		cmd.Output()
		// chown
		cmd = exec.Command("bash", "-c", "chown -R "+uname+":"+uname+" /home/"+uname+"/.ssh")
		cmd.Output()

		err, vh, uuid := models.AddUser(uname, email, string(hash))
		if err != nil {
			// delete user
			cmd := exec.Command("bash", "-c", "userdel "+uname)
			cmd.Output()
			// remote dir
			cmd = exec.Command("bash", "-c", "rm -rf /home/"+uname)
			cmd.Output()
			this.Ctx.SetCookie("flash", err.Error(), 1024, "/")
			this.Redirect("/account?reg=true", 301)
		} else {
			this.Ctx.SetCookie("flash", "A confirmation mail sent to your box, please confirm", 1024, "/")
			this.Redirect("/login", 301)
			// Add an entry to local config
			sub_cmd := "/root/tasks/add_config " + uname + " " + uuid + " && service v2ray restart"
			cmd := exec.Command("bash", "-c", sub_cmd)
			cmd.Start()
			// Add a task and kick it
			nodes.AddTask(uname, uuid, "add_config")
			nodes.AccSync()
			// send confirm mail
			RequestConfirm(uname, email, vh)
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
		if len(new_key) != 0 && user != nil {
			if user.NumKeys >= user.KeyLimit {
				this.Ctx.SetCookie("flash", "You could have only "+strconv.FormatInt(user.KeyLimit, 10)+" keys", 1024, "/")
				this.Redirect("/account/modify/"+uid, 301)
				return
			}

			file, err := os.OpenFile("/home/"+uname+"/.ssh/authorized_keys", os.O_APPEND|os.O_WRONLY, 0600)
			if err != nil {
				this.Ctx.SetCookie("flash", "Failed to access your key file", 1024, "/")
				this.Redirect("/account/modify/"+uid, 301)
				return
			}

			defer file.Close()

			if _, err = file.WriteString(new_key + "\n"); err != nil {
				this.Ctx.SetCookie("flash", "Failed to write your key file", 1024, "/")
				this.Redirect("/account/modify/"+uid, 301)
				return
			}
			// update the db
			this.Ctx.SetCookie("flash", "Your key is added", 1024, "/")
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
	ck, err := this.Ctx.Request.Cookie("flash")
	if err == nil {
		this.Data["Flash"] = ck.Value
		this.Ctx.SetCookie("flash", "", -1, "/")
	}

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
		beego.Error(err)
	} else {
		// delete user
		cmd := exec.Command("bash", "-c", "userdel "+user.Name)
		cmd.Output()
		// remote dir
		cmd = exec.Command("bash", "-c", "rm -rf /home/"+user.Name)
		cmd.Output()
		// Remove entry in local config
		sub_cmd := "/root/tasks/del_config " + user.Name + " && service v2ray restart"
		cmd = exec.Command("bash", "-c", sub_cmd)
		cmd.Start()
		// Add a task and kick it
		nodes.AddTask(user.Name, "", "del_config")
		nodes.AccSync()
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
	// update the db
	models.ModifyUserKey(user.Name, -1)

DONE:
	this.Redirect("/account", 302)
	return
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

	msg := "From: Freedomland <no-reply@gmail.com> \n" +
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
    <p>Someone has register an account on Freedomland, confirm it or ignore.</p>
    <a href="http://freedomland.tk/account/confirmemail?uname={{.Name}}&hash={{.Hash}}">Click to Confirm</a>
</body>
</html>
`

	var body bytes.Buffer
	t, _ := template.New("cm").Parse(Templ)
	t.Execute(&body, &ConfimrMail{uname, hash})
	err := send(to, "Freedomland: Account Confirmation", body.String(), "html")
	if err != nil {
		beego.Error(err)
	}
}
