package main

import (
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/logs"
	"github.com/astaxie/beego/orm"
	"github.com/robfig/cron"

	"shebao/models"
	_ "shebao/routers"
)

func init() {
	models.RegisterDB()
}

func notify_user() {
	au, err := models.AllAliveUsers()
	if err != nil {
		return
	}

	tn := time.Now()
	for _, u := range au {
		if u.Last_payed.After(tn) || u.Last_payed.Month() == tn.Month() ||
			u.Last_payed.AddDate(0, 1, 0).After(u.Expire) {
			continue
		}

		beego.Trace(u.Name, " not payed ", u.Phone)
		if len(u.Phone) == 11 {
			params := map[string][]string{
				"apikey":    {beego.AppConfig.String("yunpiankey")},
				"mobile":    {u.Phone},
				"tpl_id":    {"2224954"},
				"tpl_value": {"#name#=" + u.Name},
			}
			send_sms(params)
		} else {
			beego.Trace(u.Name, " phone number is invalid")
		}
	}
}

func notify_admin() {
	au, err := models.AllAliveUsers()
	if err != nil {
		return
	}

	tn := time.Now()
	for _, u := range au {
		if u.Last_payed.After(tn) || u.Last_payed.Month() == tn.Month() ||
			u.Last_payed.AddDate(0, 1, 0).After(u.Expire) {
			continue
		}

		beego.Trace(u.Name, " not payed !!!", u.Phone)
	}
}

func set_expire() {
	models.SetExpiredUsers()
}

func main() {
	orm.Debug = true
	orm.RunSyncdb("default", false, true)

	logs.SetLogger(logs.AdapterFile, `{"filename":"logs/freeland.log","level":7,"maxlines":0,"maxsize":0,"daily":true,"maxdays":10}`)

	// Notify user to pay on 25th
	c := cron.New()
	to_user := "0 0 0 25 * *"
	c.AddFunc(to_user, notify_user)
	to_admin := "0 0 0 1 * *"
	c.AddFunc(to_admin, notify_admin)
	user_expire := "0 0 0 * * *"
	c.AddFunc(user_expire, set_expire)
	c.Start()

	beego.Run()
}

func send_sms(params map[string][]string) error {
	header := map[string][]string{
		"Content-Type": {"application/x-www-form-urlencoded;charset=utf-8"},
	}
	req := &http.Request{
		Method: "POST",
		Header: header,
	}

	req.URL, _ = url.Parse("https://sms.yunpian.com/v2/sms/tpl_single_send.json")

	q := req.URL.Query()
	q = params
	req.URL.RawQuery = q.Encode()

	client := &http.Client{
		Timeout: 15 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	resp.Body.Close()
	beego.Trace(string(body))

	return nil
}
