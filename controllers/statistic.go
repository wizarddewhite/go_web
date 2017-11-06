package controllers

import (
	//"go_web/models"
	"encoding/json"

	"github.com/astaxie/beego"
)

type StatisticController struct {
	beego.Controller
}

func (this *StatisticController) Update() {
	var f interface{}
	err1 := json.Unmarshal(this.Ctx.Input.RequestBody, &f)
	if err1 == nil {
		m := f.(map[string]interface{})
		for _, v := range m {
			switch vv := v.(type) {
			case string:
				beego.Trace(v)
			case []interface{}:
				for _, u := range vv {
					beego.Trace(u)
					data, ok := u.(map[string]interface{})
					if ok {
						beego.Trace(data["Name"])
						beego.Trace(data["Used"])
					} else {
						beego.Trace("not match")
					}

				}
			default:
				beego.Trace("unknown type")
			}
		}
		this.Data["json"] = "{\"Status\":\"ok\"}"
		this.ServeJSON()
	} else {
		beego.Trace("unable to parse")
	}
}
