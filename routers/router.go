package routers

import (
	"go_web/controllers"

	"github.com/astaxie/beego"
)

func init() {
	beego.Router("/", &controllers.MainController{})
	beego.Router("/login", &controllers.LoginController{})
	beego.Router("/category", &controllers.CategoryController{})
	beego.Router("/topic", &controllers.TopicController{})
	beego.AutoRouter(&controllers.TopicController{})
	beego.Router("/reply", &controllers.ReplyController{})
	beego.Router("/reply/add", &controllers.ReplyController{}, "post:Add")
	beego.Router("/reply/delete", &controllers.ReplyController{}, "get:Delete")

	beego.Router("/statistic/update", &controllers.StatisticController{}, "post:Update")
	beego.Router("/account", &controllers.AccountController{})
	beego.AutoRouter(&controllers.AccountController{})

	beego.Router("/node", &controllers.NodeController{})
	beego.Router("/node/renew_k", &controllers.NodeController{}, "get:Renew")
}
