package controllers

import (
	"go_web/models"

	"github.com/astaxie/beego"
)

type TopicController struct {
	beego.Controller
}

func (this *TopicController) Get() {
	this.TplName = "topic.html"
	this.Data["Title"] = "Topic List"
	this.Data["IsTopic"] = true
	this.Data["IsLogin"], this.Data["IsAdmin"] = checkAccount(this.Ctx)
	topics, err := models.GetAllTopics("", false)
	if err != nil {
		beego.Error(err)
	} else {
		this.Data["Topics"] = topics
	}
}

func (this *TopicController) Post() {
	if login, _ := checkAccount(this.Ctx); !login {
		this.Redirect("/login", 302)
		return
	}

	title := this.Input().Get("title")
	content := this.Input().Get("content")
	tid := this.Input().Get("tid")
	category := this.Input().Get("category")
	beego.Trace(len(tid))

	var err error
	if len(tid) == 0 {
		err = models.AddTopic(title, category, content)
	} else {
		err = models.ModifyTopic(tid, title, category, content)
	}

	if err != nil {
		beego.Error(err)
	}

	this.Redirect("/topic", 302)
}

func (this *TopicController) Add() {
	this.TplName = "topic_add.html"
	this.Data["Title"] = "Add Topic"
	this.Data["IsTopic"] = true
	this.Data["IsLogin"], this.Data["IsAdmin"] = checkAccount(this.Ctx)
}

func (this *TopicController) View() {
	topic, err := models.GetTopic(this.Ctx.Input.Params()["0"])
	if err != nil {
		beego.Error(err)
		this.Redirect("/", 302)
	}

	this.TplName = "topic_view.html"
	this.Data["Title"] = topic.Title
	this.Data["IsTopic"] = true
	this.Data["IsLogin"], this.Data["IsAdmin"] = checkAccount(this.Ctx)

	this.Data["Topic"] = topic
	this.Data["Tid"] = this.Ctx.Input.Params()["0"]

	replies, err := models.GetAllReplies(this.Ctx.Input.Params()["0"])
	if err != nil {
		beego.Error(err)
		return
	}
	this.Data["Replies"] = replies
}

func (this *TopicController) Modify() {
	this.TplName = "topic_modify.html"
	tid := this.Input().Get("tid")
	topic, err := models.GetTopic(tid)
	if err != nil {
		beego.Error(err)
		this.Redirect("/", 302)
		return
	}
	this.Data["Title"] = topic.Title
	this.Data["IsTopic"] = true
	this.Data["IsLogin"], this.Data["IsAdmin"] = checkAccount(this.Ctx)
	this.Data["Topic"] = topic
	this.Data["Tid"] = tid
}

func (this *TopicController) Delete() {
	if login, _ := checkAccount(this.Ctx); !login {
		this.Redirect("/login", 302)
		return
	}

	err := models.DeleteTopic(this.Ctx.Input.Params()["0"])
	if err != nil {
		beego.Error(err)
	}
	this.Redirect("/", 302)
}
