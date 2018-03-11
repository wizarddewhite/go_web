package routers

import (
	"go_web/controllers"

	"github.com/astaxie/beego"
)

func init() {
	beego.Router("/", &controllers.MainController{})
	beego.Router("/login", &controllers.LoginController{})
	beego.Router("/forget", &controllers.ForgetController{})
	beego.Router("/reset", &controllers.ResetController{})

	beego.Router("/account", &controllers.AccountController{})
	beego.AutoRouter(&controllers.AccountController{})

	beego.Router("/permit", &controllers.PermitController{})
	beego.Router("/plan", &controllers.PlanController{})
	beego.Router("/download", &controllers.DownloadController{})
}
