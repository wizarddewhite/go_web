package controllers

import (
	"encoding/json"
	"go_web/models"
	"go_web/nodes"
	"strconv"

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
	Users string
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
				s.Users = v.(string)
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

	this.Data["json"] = "{\"Status\":\"ok\"}"
	this.ServeJSON()

	// Check the ip first
	n := nodes.GetNode(this.Ctx.Input.IP())
	if n == nil {
		beego.Trace("someone unknown send us update")
		return
	}

	// update buffer
	current_users, _ := strconv.ParseInt(s.Users, 10, 64)
	delta := int(current_users) - n.Users
	n.Users = int(current_users)
	nodes.UpdateBuffer(delta)

	for _, stat := range s.Stats {
		// write to data base
		models.ModifyUserStat(stat.Name, stat.Inbound, stat.Outbound)
		// disable a user in case out of bandwidth
	}

	// delete the node from cand_nodes in case out of bandwidth
	nodes.CheckNodeBandwidth(n)
}
