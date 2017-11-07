package controllers

import (
	"encoding/json"
	"go_web/models"

	"github.com/astaxie/beego"
)

type StatisticController struct {
	beego.Controller
}

type Stat struct {
	Name     string
	Inbound  string
	Outbound string
}

type StatSlice struct {
	Token string
	Stats []Stat
}

func (this *StatisticController) Update() {
	var f interface{}
	var s StatSlice
	err1 := json.Unmarshal(this.Ctx.Input.RequestBody, &f)
	if err1 == nil {
		m := f.(map[string]interface{})
		for _, v := range m {
			switch vv := v.(type) {
			case string:
				s.Token = v.(string)
			case []interface{}:
				for _, u := range vv {
					data, ok := u.(map[string]interface{})
					if ok {
						s.Stats = append(s.Stats,
							Stat{Name: data["Name"].(string),
								Inbound:  data["Inbound"].(string),
								Outbound: data["Outbound"].(string)})
					} else {
						beego.Trace("not match")
					}

				}
			default:
				beego.Trace("unknown type")
			}
		}
	} else {
		beego.Trace("unable to parse")
		return
	}

	// Check the token first
	beego.Trace(s.Token)

	this.Data["json"] = "{\"Status\":\"ok\"}"
	this.ServeJSON()
	for _, stat := range s.Stats {
		//beego.Trace(stat)
		// check bandwidth first
		// write to data base

		models.ModifyUserStat(stat.Name, stat.Inbound, stat.Outbound)
	}
}
