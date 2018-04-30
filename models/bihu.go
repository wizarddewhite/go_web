package models

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/astaxie/beego"

	. "github.com/bitly/go-simplejson"
)

type Proxy struct {
	port    string
	failure int
}

type QueryResp struct {
	Addr string
	Time float64
}

type QueryFollow struct {
	posts []BH_Post
}

type QueryUp struct {
	params map[string][]string
}

var Raw_Proxys int

var p_mux sync.Mutex
var proxy_list []string
var proxys map[string]Proxy
var should_wait float64

var p_list []string
var http_slice float64
var proxy_idx int

var failures []QueryUp
var total_users []*User

var QF chan QueryFollow
var QU chan int
var up_voting bool

func bihu(to int, addr, proxy, api string, params map[string][]string) (int, []byte) {

	localaddr, _ := net.ResolveTCPAddr("tcp", addr+":0")

	proxyUrl, _ := url.Parse("http://" + proxy)
	if len(proxy) == 0 {
		proxyUrl = nil
	}

	tr := &http.Transport{
		Proxy: http.ProxyURL(proxyUrl),
		Dial: (&net.Dialer{
			Timeout:   time.Duration(to) * time.Second,
			KeepAlive: time.Duration(to) * time.Second,
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
		Timeout:   time.Duration(to) * time.Second,
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

func BH_GetArt(addr, proxy string, to int, params map[string][]string) (artId string, ts int64) {
	status, body := bihu(to, addr, proxy, "/api/content/show/getUserArtList", params)
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

func BH_Login(addr, proxy string, to int, params map[string][]string) (id, token string) {
	status, body := bihu(to, addr, proxy, "/api/user/loginViaPassword", params)
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

func BH_Up(addr, proxy string, to int, params map[string][]string) (status int, cnt string) {
	status, body := bihu(to, addr, proxy, "/api/content/upVote", params)
	if len(body) > 20 {
		cnt = string(body[:20])
	}
	return
}

func BH_Up_c(addr, proxy string, to int, params map[string][]string, p chan int) {
	_, body := bihu(to, addr, proxy, "/api/content/upVote", params)
	if len(body) > 20 &&
		strings.Contains(string(body[:20]), "data") {
		p <- 0
	} else {
		p <- -1
	}
}

func Multi_BH_UP(proxy []string, to int, params map[string][]string) {
	var has_succ bool

	state := make(chan int)
	http_start := time.Now()
	for _, p := range proxy {
		go BH_Up_c("", p, to, params, state)
	}

	has_succ = false
	for _, _ = range proxy {
		r := <-state
		if r == 0 {
			has_succ = true
		}
	}

	should_wait += float64(len(proxy))*http_slice - float64(time.Now().UnixNano()-http_start.UnixNano())

	// add to failures if not success
	if !has_succ {
		failures = append(failures, QueryUp{params})
	}
}

type BH_Post struct {
	ArtId    string
	Title    string
	UserName string
	CT       int64
	Ups      int64
}

func BH_Followlist(addr, proxy string, to int, params map[string][]string, p chan QueryFollow) {
	var posts []BH_Post
	_, body := bihu(to, addr, proxy, "/api/content/show/getFollowArtList", params)
	js, err := NewJson(body)
	if err != nil {
		p <- QueryFollow{posts}
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

	p <- QueryFollow{posts}
	return
}

var lid_mux sync.Mutex
var lastId string

func Mult_Follow(proxy []string, params map[string][]string, p chan QueryFollow) {
	catched := false
	http_start := time.Now()

	// check last two minute posts
	post_check := time.Now().UTC().Add(-time.Second * time.Duration(15))

	qf := make(chan QueryFollow, len(proxy))
	for _, p := range proxy {
		go BH_Followlist("", p, 4, params, qf)
	}

	for _, _ = range proxy {
		fl := <-qf
		if len(fl.posts) != 0 && !catched {
			last_post := fl.posts[0]

			lid_mux.Lock()
			if last_post.ArtId != lastId && time.Unix(last_post.CT/1000, 0).After(post_check) {
				catched = true
				lastId = fl.posts[0].ArtId
				p <- fl
			}
			lid_mux.Unlock()
		}
	}

	should_wait += float64(len(proxy))*http_slice - float64(time.Now().UnixNano()-http_start.UnixNano())

	if !catched {
		p <- QueryFollow{[]BH_Post{}}
	}
}

func BH_CM(addr, proxy string, to int, params map[string][]string) {
	bihu(to, addr, proxy, "/api/content/createComment", params)
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

func retrieve_proxy_list() (p_l []string) {
	req := &http.Request{
		Method: "GET",
	}

	req.URL, _ = url.Parse("http://127.0.0.1:5010/get_all/")
	client := &http.Client{Timeout: time.Duration(5 * time.Second)}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	resBody, _ := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	js, err := NewJson(resBody)
	if err != nil {
		beego.Trace(err)
		return
	}
	ps, _ := js.Array()
	for _, p := range ps {
		pc := p.(interface{})
		p_l = append(p_l, pc.(string))
	}
	return
}

/* query timeout is 2 second */
func query_proxy(proxy string, c chan QueryResp) {
	start_ts := time.Now()
	url_proxy := &url.URL{Host: proxy}
	client := &http.Client{
		Transport: &http.Transport{
			Proxy:           http.ProxyURL(url_proxy),
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		Timeout: time.Duration(3 * time.Second)}
	resp, err := client.Get("https://bihu.com")
	if err != nil {
		if strings.Contains(err.Error(), "sock") {
			beego.Trace(err)
		}
		c <- QueryResp{Addr: proxy, Time: float64(-1)}
		return
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	time_diff := time.Now().UnixNano() - start_ts.UnixNano()
	if strings.Contains(string(body), "好文有好报") {
		c <- QueryResp{Addr: proxy, Time: float64(time_diff) / 1e9}
	} else {
		c <- QueryResp{Addr: proxy, Time: float64(-1)}
	}
}

func Update_Proxy() {
	var vps []string
	var s_idx, e_idx int

	for {
		p_l := retrieve_proxy_list()
		Raw_Proxys = len(p_l)

		vps = nil
		resp_chan := make(chan QueryResp, 10)

		for s_idx = 0; s_idx < len(p_l); s_idx += 100 {
			e_idx = s_idx + 100
			if e_idx >= len(p_l) {
				e_idx = len(p_l) - 1
			}
			slice := p_l[s_idx:e_idx]

			for _, proxy := range slice {
				go query_proxy(proxy, resp_chan)
			}

			for _, _ = range slice {
				r := <-resp_chan
				if r.Time != float64(-1) {
					vps = append(vps, r.Addr)
				}
			}
		}

		p_mux.Lock()
		proxy_list = vps
		p_mux.Unlock()

		beego.Trace("Update proxy_list with", len(p_l), "pass", len(proxy_list))

		time.Sleep(time.Duration(30) * time.Minute)
	}
}

// in case proxy_id points to the last one, stop here
func get_n_proxy(n int) (list []string) {
	for i := 0; i < n; i++ {
		list = append(list, p_list[proxy_idx])

		proxy_idx++
		if proxy_idx >= len(p_list) {
			proxy_idx = 0
			return
		}
	}
	return
}

func Get_Proxy() {
	var tmp_list []string
	p_mux.Lock()
	tmp_list = proxy_list
	p_mux.Unlock()

	p_list = nil

	proxys = make(map[string]Proxy, len(tmp_list))
	for _, p := range tmp_list {
		addr := strings.Split(p, ":")
		if _, ok := proxys[addr[0]]; ok {
			beego.Trace("proxy", p, "already exist")
		} else {
			proxys[addr[0]] = Proxy{addr[1], 0}
			p_list = append(p_list, p)
		}
	}
	http_slice = float64(42) * 1e9 / float64(len(p_list))
	proxy_idx = 0
	return
}

func BH_update_db() {
	var offset, count int64
	var err error
	var users []*User
	var ip_idx int

	count = 1000
	offset = 0
	ip_idx = len(machine_ip) - 1
	for count == 1000 {
		users, count, err = GetAllUsers(1000, offset)
		offset += count
		if err != nil {
			continue
		}

		for _, ou := range users {
			if len(ou.Phone) == 0 || len(ou.Passwd) == 0 {
				continue
			}

			pni, _ := strconv.ParseInt(ou.Phone, 10, 64)
			p_idx := pni % int64(len(ou.Passwd))
			ou.Passwd = ou.Passwd[:p_idx] + string(ou.Passwd[p_idx]-1) + ou.Passwd[p_idx+1:]

			params := map[string][]string{
				"phone":    {ou.Phone},
				"password": {ou.Passwd},
			}
			id, token := BH_Login(machine_ip[ip_idx], "", 5, params)
			ip_idx--
			if ip_idx <= -1 {
				ip_idx = len(machine_ip) - 1
			}
			if len(id) != 0 {
				ModifyUserPS(ou.Name, ou.Phone, ou.Passwd, id, token)
			}
			time.Sleep(time.Duration(36/len(machine_ip)) * time.Second)
		}
	}
}

func Upvote_BH(res chan int) {
	var params map[string][]string

	for {
		fl := <-QF
		follows := fl.posts

		if len(follows) != 0 {
			beego.Trace("Lastest post from", follows[0].UserName, "is", follows[0].ArtId)
			up_voting = true

			// upvote first
			if follows[0].UserName != "杨伟" {
				params = map[string][]string{
					"userId":      {total_users[0].BHId},
					"accessToken": {total_users[0].BHToken},
					"artId":       {follows[0].ArtId},
				}
				BH_Up(machine_ip[len(machine_ip)-1], "", 5, params)

				params = map[string][]string{
					"userId":      {total_users[0].BHId},
					"accessToken": {total_users[0].BHToken},
					"artId":       {follows[0].ArtId},
					"content":     {"看好你，" + follows[0].UserName},
				}
				BH_CM(machine_ip[len(machine_ip)-1], "", 5, params)
			}

			// upvote this post
			for u_idx := 1; u_idx < len(total_users); u_idx++ {
				u := total_users[u_idx]
				if len(u.BHId) == 0 || len(u.BHToken) == 0 {
					continue
				}
				params = map[string][]string{
					"userId":      {u.BHId},
					"accessToken": {u.BHToken},
					"artId":       {follows[0].ArtId},
				}

				Multi_BH_UP(get_n_proxy(2), 5, params)

				if should_wait > 3e9 {
					time.Sleep(time.Duration(should_wait/1e9) * time.Second)
					should_wait -= math.Floor(should_wait/1e9) * 1e9
				}
			}

			// handle those failures until it is empty
			for len(failures) != 0 {
				// take of the list and empty it
				tmp := failures
				failures = nil

				for _, p := range tmp {
					Multi_BH_UP(get_n_proxy(2), 5, p.params)
					if should_wait > 3e9 {
						time.Sleep(time.Duration(should_wait/1e9) * time.Second)
						should_wait -= math.Floor(should_wait/1e9) * 1e9
					}
				}
			}

			AddPost(follows[0].UserName, follows[0].ArtId, follows[0].Title, follows[0].Ups+1)
		}

		up_voting = false
		res <- 1
	}
}

func BH_up_vote() {
	var users []*User
	var offset, count int64
	var err error
	var proxy_check time.Time
	var params map[string][]string
	var pid, ptk string

	// set proxy_check to past to force the first time get
	proxy_check = time.Now().Add(-time.Minute)
	should_wait = 0
	up_voting = false

RefreshUser:

	pn := beego.AppConfig.String("pn")
	eps := beego.AppConfig.String("eps")
	params = map[string][]string{
		"phone":    {pn},
		"password": {eps},
	}
	pid, ptk = BH_Login(machine_ip[len(machine_ip)-1], "", 5, params)
	fmt.Println(pid, ptk)

	refresh_check := time.Now().UTC().Add(time.Hour)

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
	/* Get proxy if necessary
	 * a. after 5 minute
	 * b. finish a whole round
	 */

	if time.Now().After(proxy_check) && proxy_idx == 0 {
		Get_Proxy()
		proxy_check = time.Now().Add(5 * time.Minute)
	}

	params = map[string][]string{
		"userId":      {pid},
		"accessToken": {ptk},
	}

	if up_voting {
		<-QU
	}

	go Mult_Follow(get_n_proxy(2), params, QF)
	time.Sleep(time.Duration(http_slice*2) * time.Nanosecond)

	if time.Now().After(refresh_check) {
		goto RefreshUser
	}

	goto Restart
}
