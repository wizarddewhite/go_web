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
	"strings"
	"sync"
	"time"

	"github.com/astaxie/beego"

	. "github.com/bitly/go-simplejson"
)

type QueryResp struct {
	Addr string
	Time float64
}

var p_mux sync.Mutex
var proxy_list []string
var proxys map[string]int

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

type BH_Post struct {
	ArtId    string
	Title    string
	UserName string
	CT       int64
	Ups      int64
}

func BH_Followlist(addr, proxy string, to int, params map[string][]string) (posts []BH_Post) {
	_, body := bihu(to, addr, proxy, "/api/content/show/getFollowArtList", params)
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

/* query timeout is 3 second */
func query_proxy(proxy string, c chan QueryResp) {
	start_ts := time.Now()
	url_proxy := &url.URL{Host: proxy}
	client := &http.Client{
		Transport: &http.Transport{Proxy: http.ProxyURL(url_proxy)},
		Timeout:   time.Duration(3 * time.Second)}
	resp, err := client.Get("https://bihu.com")
	if err != nil {
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
	for {
		p_l := retrieve_proxy_list()

		resp_chan := make(chan QueryResp, 10)
		for _, proxy := range p_l {
			go query_proxy(proxy, resp_chan)
		}

		vps = nil

		for _, _ = range p_l {
			r := <-resp_chan
			if r.Time > 1e-9 {
				vps = append(vps, r.Addr)
			}
		}

		p_mux.Lock()
		proxy_list = vps
		p_mux.Unlock()

		beego.Trace("Update proxy_list with", len(proxy_list))

		time.Sleep(time.Duration(1) * time.Minute)
	}
}

func Get_Proxy() (p_l []string) {
	p_mux.Lock()
	p_l = proxy_list
	p_mux.Unlock()
	return
}

func BH_up_vote() {
	var total_users []*User
	var users []*User
	var offset, count int64
	var err error
	var post_check time.Time
	var params map[string][]string
	var pid, ptk string
	var ip_idx int

	ip_idx = len(machine_ip) - 1

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

	// check last two minute posts
	post_check = time.Now().UTC().Add(-time.Minute * time.Duration(2))

	params = map[string][]string{
		"userId":      {pid},
		"accessToken": {ptk},
	}
	follows := BH_Followlist(machine_ip[ip_idx], "", 5, params)
	ip_idx--
	if ip_idx <= -1 {
		ip_idx = len(machine_ip) - 1
	}

	if len(follows) != 0 && time.Unix(follows[0].CT/1000, 0).After(post_check) {
		beego.Trace("Lastest post from", follows[0].UserName, "is", follows[0].ArtId)
		// upvote this post
		for u_idx := 0; u_idx < len(total_users); u_idx++ {
			u := total_users[u_idx]
			if len(u.BHId) == 0 || len(u.BHToken) == 0 {
				continue
			}
			params = map[string][]string{
				"userId":      {u.BHId},
				"accessToken": {u.BHToken},
				"artId":       {follows[0].ArtId},
			}
			BH_Up(machine_ip[ip_idx], "", 5, params)

			if u.BHId == "179159" {
				params = map[string][]string{
					"userId":      {u.BHId},
					"accessToken": {u.BHToken},
					"artId":       {follows[0].ArtId},
					"content":     {"看好你，" + follows[0].UserName},
				}
				BH_CM(machine_ip[ip_idx], "", 5, params)
			}
			time.Sleep(time.Duration(36/len(machine_ip)) * time.Second)
			ip_idx--
			if ip_idx <= -1 {
				ip_idx = len(machine_ip) - 1
			}
		}

		AddPost(follows[0].UserName, follows[0].ArtId, follows[0].Title, follows[0].Ups+1)
	}

	time.Sleep(time.Duration(42/len(machine_ip)) * time.Second)

	if time.Now().After(refresh_check) {
		goto RefreshUser
	}

	goto Restart
}
