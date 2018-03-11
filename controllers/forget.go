package controllers

import (
	"bytes"
	"text/template"

	"go_web/models"

	"github.com/astaxie/beego"
	//"github.com/astaxie/beego/context"
)

type ForgetController struct {
	beego.Controller
}

func (this *ForgetController) Get() {
	this.Data["Title"] = "Forget Password"
	this.TplName = "forget_pass.html"
	return
}

func (this *ForgetController) Post() {
	uname := this.Input().Get("uname")

	Templ := `
<html>
<body>
    <h3>Reset Password:</h3>
    <p>Someone has request password reset, click the link or ignore.</p>
    <a href="http://GapSeeker.tk/reset?uname={{.Name}}&hash={{.Hash}}">Click to Confirm</a>
</body>
</html>
`

	email, hash := models.GetUserResetHash(uname)
	if len(hash) == 0 {
		this.Ctx.SetCookie("flash", "No such user", 1024, "/")
		this.Redirect("/login", 301)
		return
	}
	var body bytes.Buffer
	t, _ := template.New("cm").Parse(Templ)
	t.Execute(&body, &ConfimrMail{uname, hash})
	send(email, "Webframe: Reset Password", body.String(), "html")
	this.Ctx.SetCookie("flash", "Password reset link has sent to your mail", 1024, "/")
	this.Redirect("/", 301)
	return
}
