package routers

import (
	"bihu_helper/controllers"

	"github.com/astaxie/beego"
)

func init() {
	beego.Router("/", &controllers.MainController{})
	beego.Router("/login", &controllers.LoginController{})
	beego.Router("/forget", &controllers.ForgetController{})
	beego.Router("/reset", &controllers.ResetController{})

	beego.Router("/account", &controllers.AccountController{})
	beego.AutoRouter(&controllers.AccountController{})

	beego.Router("/tutorial", &controllers.TutorialController{})
	beego.Router("/status", &controllers.StatusController{})
}
