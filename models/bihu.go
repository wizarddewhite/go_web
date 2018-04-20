package models

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"sort"
	"strconv"
	"time"

	"github.com/astaxie/beego"

	. "github.com/bitly/go-simplejson"
)

func bihu(addr, api string, params map[string][]string) (int, []byte) {
	localaddr, _ := net.ResolveTCPAddr("tcp", addr+":0")
	tr := &http.Transport{
		Dial: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			LocalAddr: localaddr}).Dial,
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	req := &http.Request{
		Method: "POST",
	}

	req.URL, _ = url.Parse("https://be02.bihu.com/bihube-pc" + api)
	q := req.URL.Query()
	q = params
	req.URL.RawQuery = q.Encode()

	client := &http.Client{
		Timeout:   15 * time.Second,
		Transport: tr,
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, nil
	}

	resBody, _ := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	return resp.StatusCode, resBody
}

func BH_GetArt(addr string, params map[string][]string) (artId string, ts int64) {
	status, body := bihu(addr, "/api/content/show/getUserArtList", params)
	if status != http.StatusOK {
		return "", 0
	}
	js, err := NewJson(body)
	if err != nil {
		return "", 0
	}
	arts, err := js.Get("data").Get("list").Array()
	if err != nil {
		return "", 0
	}
	art := arts[0].(map[string]interface{})
	id, _ := art["id"].(json.Number).Int64()
	ts, _ = art["createTime"].(json.Number).Int64()
	artId = strconv.FormatInt(id, 10)
	return
}

func BH_Login(addr string, params map[string][]string) (id, token string) {
	status, body := bihu(addr, "/api/user/loginViaPassword", params)
	if status != http.StatusOK {
		return "", ""
	}
	js, err := NewJson(body)
	if err != nil {
		return "", ""
	}
	id, err = js.Get("data").Get("userId").String()
	token, err = js.Get("data").Get("accessToken").String()
	return
}

func BH_Up(addr string, params map[string][]string) (status int) {
	status, _ = bihu(addr, "/api/content/upVote", params)
	return
}

type BH_Post struct {
	ArtId    string
	Title    string
	UserName string
	CT       int64
	Ups      int64
}

func BH_Followlist(addr string, params map[string][]string) (posts []BH_Post) {
	_, body := bihu(addr, "/api/content/show/getFollowArtList", params)
	js, err := NewJson(body)
	if err != nil {
		return
	}
	ps, err := js.Get("data").Get("artList").Get("list").Array()
	for _, p := range ps {
		pc := p.(map[string]interface{})
		id, _ := pc["id"].(json.Number).Int64()
		title, _ := pc["title"].(string)
		un, _ := pc["userName"].(string)
		ts, _ := pc["createTime"].(json.Number).Int64()
		ups, _ := pc["ups"].(json.Number).Int64()
		posts = append(posts, BH_Post{strconv.FormatInt(id, 10), title, un, ts, ups})
	}
	return
}

func BH_CM(addr string, params map[string][]string) {
	bihu(addr, "/api/content/createComment", params)
	return
}

var machine_ip []string

type Sibling struct {
	ip string
	op string
}

var siblings = []Sibling{
	{"45.32.185.10", "-P 26 "},
}

func BH_retrieve_ip() {
	ifaces, _ := net.Interfaces()
	// handle err
	for _, i := range ifaces {
		addrs, _ := i.Addrs()
		// handle err
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
				if !v.IP.IsLoopback() && v.IP.To4() != nil {
					machine_ip = append(machine_ip, ip.String())
				}
			}
			// process IP address
		}
	}
}

func send_data(s Sibling) {
	file := "/root/go/src/bihu_helper/data/beeblog.db "
	cmd := exec.Command("bash", "-c",
		"scp "+s.op+file+"root@"+s.ip+":"+file)
	_, err := cmd.Output()
	if err != nil {
		beego.Trace(err)
	}
}

func BH_up_vote() {
	var total_users []*User
	var users []*User
	var offset, count int64
	var err error
	var t1 time.Time
	var params map[string][]string
	var pid, ptk string
	var ip_idx int

	ip_idx = len(machine_ip) - 1
	time.Sleep(time.Duration(10) * time.Second)

RefreshUser:
	// transfer data to remote
	for _, s := range siblings {
		send_data(s)
	}

	pn := beego.AppConfig.String("pn")
	eps := beego.AppConfig.String("eps")
	params = map[string][]string{
		"phone":    {pn},
		"password": {eps},
	}
	pid, ptk = BH_Login(machine_ip[len(machine_ip)-1], params)
	fmt.Println(pid, ptk)

	ts := time.Now().UTC()

	count = 1000
	offset = 0

	// retrieve all users
	total_users = nil
	for count == 1000 {
		users, count, err = GetAllUsers(1000, offset)
		offset += count
		if err != nil {
			continue
		}

		// adjust content
		for _, ou := range users {
			if len(ou.Passwd) == 0 || len(ou.BHToken) == 0 {
				continue
			}

			pni, _ := strconv.ParseInt(ou.Phone, 10, 64)
			p_idx := pni % int64(len(ou.Passwd))
			ou.Passwd = ou.Passwd[:p_idx] + string(ou.Passwd[p_idx]-1) + ou.Passwd[p_idx+1:]
			t_idx := pni % int64(len(ou.BHToken))
			ou.BHToken = ou.BHToken[:t_idx] + string(ou.BHToken[t_idx]-1) + ou.BHToken[t_idx+1:]
		}

		total_users = append(total_users, users...)
	}

	// order by Recommends
	sort.Slice(total_users, func(i, j int) bool {
		return total_users[i].Recommends > total_users[j].Recommends
	})

Restart:

	// check last two minute posts
	t1 = time.Now().UTC().Add(-time.Minute * time.Duration(2))

	params = map[string][]string{
		"userId":      {pid},
		"accessToken": {ptk},
	}
	follows := BH_Followlist(machine_ip[ip_idx], params)
	ip_idx--
	if ip_idx <= -1 {
		ip_idx = len(machine_ip) - 1
	}

	for _, p := range follows {

		//time.Sleep(time.Minute)

		beego.Trace("Lastest post from", p.UserName, "is", p.ArtId)

		// skip an old post
		if t1.After(time.Unix(p.CT/1000, 0)) {
			break
		}

		// invoke remote action

		// upvote this post
		for u_idx := 0; u_idx < len(total_users); u_idx++ {
			u := total_users[u_idx]
			if len(u.BHId) == 0 || len(u.BHToken) == 0 {
				continue
			}
			params = map[string][]string{
				"userId":      {u.BHId},
				"accessToken": {u.BHToken},
				"artId":       {p.ArtId},
			}
			BH_Up(machine_ip[ip_idx], params)

			if u.BHId == "179159" {
				params = map[string][]string{
					"userId":      {u.BHId},
					"accessToken": {u.BHToken},
					"artId":       {p.ArtId},
					"content":     {"看好你，" + p.UserName},
				}
				BH_CM(machine_ip[ip_idx], params)
			}
			time.Sleep(time.Duration(36/len(machine_ip)) * time.Second)
			ip_idx--
			if ip_idx <= -1 {
				ip_idx = len(machine_ip) - 1
			}
		}

		AddPost(p.UserName, p.ArtId, p.Title, p.Ups+1)
		break
	}

	tn := time.Now().UTC()
	elapsed := tn.Sub(ts)
	if elapsed > time.Hour {
		goto RefreshUser
	}

	time.Sleep(time.Duration(42/len(machine_ip)) * time.Second)

	goto Restart
}
