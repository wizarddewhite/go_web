package controllers

import (
	"github.com/astaxie/beego"
)

type MainController struct {
	beego.Controller
}

func (c *MainController) Get() {
	c.TplName = "index.tpl"
	beego.Trace("test trace")

	c.Data["TrueCond"] = true
	c.Data["FalseCond"] = false

	type u struct {
		Name string
		Age  int
		Sex  string
	}

	user := &u{
		Name: "Joe",
		Age:  20,
		Sex:  "Male",
	}
	c.Data["User"] = user

	nums := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 0}
	c.Data["Nums"] = &nums
}
