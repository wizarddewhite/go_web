package controllers

import (
	"github.com/astaxie/beego"
)

type DownloadController struct {
	beego.Controller
}

func (this *DownloadController) Get() {
	this.TplName = "download.html"
	this.Data["Title"] = "Download"
	this.Data["IsDownload"] = true
	getLoginUser(&this.Controller)
}
