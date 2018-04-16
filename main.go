package main

import (
	"fmt"
	"sort"
	"time"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/logs"
	"github.com/astaxie/beego/orm"
	//"github.com/robfig/cron"

	"bihu_helper/models"
	_ "bihu_helper/routers"
)

func init() {
	models.RegisterDB()
}

var pop_star = []string{
	"179159", // me
	"9909",   // jinma
	"1385",   // 爱思考的糖
	"131507", // 圊呓语
	"483",    // 玩火的猴子
	"2234",   // 南宫远
	"11880",  // 湘乡的大树
	"55332",  // 吴庆英
	"12627",  // jimi
	"193646", // wdctll
	"9457",   // Bean
	"13599",  // 陈竹
}

func up_vote() {
	var total_users []*models.User
	var users []*models.User
	var offset, count int64
	var err error
	var t1, t2 time.Time
	var params map[string][]string
	var pid, ptk string

	time.Sleep(time.Duration(10) * time.Second)

	// check last two minute posts
	t1 = time.Now().UTC().Add(-time.Minute * time.Duration(2))

RefreshUser:
	pn := beego.AppConfig.String("pn")
	eps := beego.AppConfig.String("eps")
	params = map[string][]string{
		"phone":    {pn},
		"password": {eps},
	}
	pid, ptk = models.BH_Login(params)
	fmt.Println(pid, ptk)

	ts := time.Now().UTC()

	count = 1000
	offset = 0

	// retrieve all users
	for count == 1000 {
		users, count, err = models.GetAllUsers(1000, offset)
		offset += count
		if err != nil {
			continue
		}

		total_users = append(total_users, users...)
	}

	// order by Recommends
	sort.Slice(total_users, func(i, j int) bool {
		return total_users[i].Recommends > total_users[j].Recommends
	})

Restart:

	t2 = time.Now().UTC()

	for _, starId := range pop_star {

		time.Sleep(time.Duration(36) * time.Second)

		// retrieve the latest post
		params = map[string][]string{
			"userId":      {pid},
			"accessToken": {ptk},
			"queryUserId": {starId},
			"pageNum":     {"1"},
		}

		artId, ts := models.BH_GetArt(params)
		beego.Trace("Lastest post from", starId, "is", artId)

		// skip an old post
		if t1.After(time.Unix(ts/1000, 0)) {
			continue
		}

		// upvote this post
		for _, u := range total_users {
			if len(u.BHId) == 0 || len(u.BHToken) == 0 {
				continue
			}
			time.Sleep(time.Duration(2) * time.Second)
			params = map[string][]string{
				"userId":      {u.BHId},
				"accessToken": {u.BHToken},
				"artId":       {artId},
			}
			models.BH_Up(params)
		}
	}

	// adjust the time mark
	t1 = t2

	tn := time.Now().UTC()
	elapsed := tn.Sub(ts)
	if elapsed > time.Hour {
		goto RefreshUser
	}

	goto Restart
}

func main() {
	orm.Debug = true
	orm.RunSyncdb("default", false, true)

	logs.SetLogger(logs.AdapterFile, `{"filename":"logs/freeland.log","level":7,"maxlines":0,"maxsize":0,"daily":true,"maxdays":10}`)

	go up_vote()

	beego.Run()
}
