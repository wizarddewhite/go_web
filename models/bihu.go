package models

import (
	"crypto/tls"
	"encoding/json"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"

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
