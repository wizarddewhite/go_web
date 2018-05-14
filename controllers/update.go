package controllers

import (
	"encoding/json"

	"github.com/astaxie/beego"

	"bihu_helper/models"
)

type UpdateController struct {
	beego.Controller
}

func (this *UpdateController) Post() {
	var p models.Post
	err := json.Unmarshal(this.Ctx.Input.RequestBody, &p)
	if err != nil {
		this.Data["json"] = "{\"Status\":\"fail\"}"
		this.ServeJSON()
		return
	}
	this.Data["json"] = "{\"Status\":\"ok\"}"
	this.ServeJSON()
	models.AddPost(p.UserN, p.ArtId, p.Title, p.Ups)
	return
}
