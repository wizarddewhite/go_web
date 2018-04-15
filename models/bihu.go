package models

import (
	"crypto/tls"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"

	. "github.com/bitly/go-simplejson"
)

func bihu(api string, params map[string][]string) (int, []byte) {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	req := &http.Request{
		Method: "POST",
	}

	req.URL, _ = url.Parse("https://be02.bihu.com/bihube-pc" + api)
	q := req.URL.Query()
	q = params
	req.URL.RawQuery = q.Encode()

	client := &http.Client{Transport: tr}
	resp, err := client.Do(req)
	if err != nil {
		return 0, nil
	}

	resBody, _ := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	return resp.StatusCode, resBody
}

func BH_GetArt(params map[string][]string) (artId string, ts int64) {
	status, body := bihu("/api/content/show/getUserArtList", params)
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

func BH_Login(params map[string][]string) (id, token string) {
	status, body := bihu("/api/user/loginViaPassword", params)
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

func BH_Up(params map[string][]string) (status int) {
	status, _ = bihu("/api/content/upVote", params)
	return
}