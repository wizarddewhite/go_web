package models

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
	"path"
	"strconv"
	"time"

	"github.com/Unknwon/com"
	"github.com/astaxie/beego/orm"
	_ "github.com/mattn/go-sqlite3"
)

const (
	_DB_NAME        = "data/beeblog.db"
	_SQLITE3_DRIVER = "sqlite3"
)

type Category struct {
	Id              int64
	Title           string
	Created         time.Time `orm:"index"`
	Views           int64     `orm:"index"`
	TopicTime       time.Time
	TopicCount      int64
	TopicLastUserId int64
}

func AddCategory(name string) error {
	o := orm.NewOrm()

	cate := &Category{
		Title:     name,
		Created:   time.Now(),
		TopicTime: time.Now(),
	}

	qs := o.QueryTable("category")
	err := qs.Filter("title", name).One(cate)
	if err == nil {
		return err
	}

	_, err = o.Insert(cate)
	if err != nil {
		return err
	}
	return nil
}

func DeleteCategory(id string) error {
	cid, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return err
	}

	o := orm.NewOrm()
	cate := &Category{Id: cid}
	_, err = o.Delete(cate)
	return err
}

func GetAllCategories() ([]*Category, error) {
	o := orm.NewOrm()

	cates := make([]*Category, 0)
	qs := o.QueryTable("category")
	_, err := qs.All(&cates)
	return cates, err
}

type Topic struct {
	Id              int64
	Uid             int64
	Title           string
	Category        string
	Content         string `orm:"size(5000)"`
	Attachment      string
	Created         time.Time `orm:"index"`
	Updated         time.Time `orm:"index"`
	Views           int64     `orm:"index"`
	Author          string
	ReplyTime       time.Time `orm:"index"`
	ReplyCount      int64
	ReplyLastUserId int64
}

func AddTopic(title, category, content string) error {
	o := orm.NewOrm()

	topic := &Topic{
		Title:     title,
		Content:   content,
		Category:  category,
		Created:   time.Now(),
		Updated:   time.Now(),
		ReplyTime: time.Now(),
	}

	_, err := o.Insert(topic)
	return err
}

func GetAllTopics(cate string, isDesc bool) ([]*Topic, error) {
	o := orm.NewOrm()

	topics := make([]*Topic, 0)
	qs := o.QueryTable("topic")

	var err error
	if isDesc {
		if len(cate) > 0 {
			qs = qs.Filter("category", cate)
		}
		_, err = qs.OrderBy("-created").All(&topics)
	} else {
		_, err = qs.All(&topics)
	}
	return topics, err
}

func GetTopic(tid string) (*Topic, error) {
	tidNum, err := strconv.ParseInt(tid, 10, 64)
	if err != nil {
		return nil, err
	}

	o := orm.NewOrm()

	topic := new(Topic)
	qs := o.QueryTable("topic")
	err = qs.Filter("id", tidNum).One(topic)
	if err != nil {
		return nil, err
	}
	topic.Views++
	_, err = o.Update(topic)
	return topic, err
}

func ModifyTopic(tid, title, category, content string) error {
	tidNum, err := strconv.ParseInt(tid, 10, 64)
	if err != nil {
		return err
	}

	o := orm.NewOrm()
	topic := &Topic{Id: tidNum}
	if o.Read(topic) == nil {
		topic.Title = title
		topic.Content = content
		topic.Category = category
		topic.Updated = time.Now()
		o.Update(topic)
		return nil
	}
	return errors.New("can't work with 42")
}

func DeleteTopic(tid string) error {
	tidNum, err := strconv.ParseInt(tid, 10, 64)
	if err != nil {
		return err
	}

	o := orm.NewOrm()
	topic := &Topic{Id: tidNum}
	_, err = o.Delete(topic)
	return err
}

type Comment struct {
	Id      int64
	Tid     int64
	Name    string
	Content string    `orm:"size(1000)"`
	Created time.Time `orm:"index"`
}

func AddReply(tid, nickname, content string) error {
	tidNum, err := strconv.ParseInt(tid, 10, 64)
	if err != nil {
		return err
	}

	reply := &Comment{
		Tid:     tidNum,
		Name:    nickname,
		Content: content,
		Created: time.Now(),
	}

	o := orm.NewOrm()
	_, err = o.Insert(reply)
	return err
}

func DeleteReply(rid string) error {
	ridNum, err := strconv.ParseInt(rid, 10, 64)
	if err != nil {
		return err
	}
	o := orm.NewOrm()
	reply := &Comment{Id: ridNum}
	_, err = o.Delete(reply)
	return err
}

func GetAllReplies(tid string) ([]*Comment, error) {
	tidNum, err := strconv.ParseInt(tid, 10, 64)
	if err != nil {
		return nil, err
	}

	replies := make([]*Comment, 0)
	o := orm.NewOrm()
	qs := o.QueryTable("comment")
	_, err = qs.Filter("tid", tidNum).All(&replies)
	return replies, err
}

type User struct {
	Id      int64
	Name    string `orm:"index"`
	Email   string
	VHash   string
	IsAdmin bool
	PWD     string
	UUID    string

	// bandwidth
	Total    float64
	Inbound  float64
	Outbound float64

	// expire
	Expire     time.Time
	NextRefill time.Time

	// key manage
	KeyLimit int64 `orm:"default(2)"`
	NumKeys  int64 `orm:"default(0)"`
}

func DeleteUser(id string) error {
	uid, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return err
	}

	o := orm.NewOrm()
	user := &User{Id: uid}
	_, err = o.Delete(user)
	return err
}

var mark_t = time.Date(2009, 11, 17, 20, 34, 58, 0, time.UTC)
var check_t = time.Date(2009, 11, 17, 20, 34, 59, 0, time.UTC)
var byteGroups = []int{8, 4, 4, 4, 12}

type UUID [16]byte

func (u *UUID) Bytes() []byte {
	return u[:]
}

func (u *UUID) String() string {
	bytes := u.Bytes()
	result := hex.EncodeToString(bytes[0 : byteGroups[0]/2])
	start := byteGroups[0] / 2
	for i := 1; i < len(byteGroups); i++ {
		nBytes := byteGroups[i] / 2
		result += "-"
		result += hex.EncodeToString(bytes[start : start+nBytes])
		start += nBytes
	}
	return result
}

func AddUser(name, email, pwd string) (error, string) {
	o := orm.NewOrm()

	user := &User{
		Name:       name,
		PWD:        pwd,
		Email:      email,
		Expire:     mark_t,
		NextRefill: mark_t,
		KeyLimit:   2,
	}

	qs := o.QueryTable("user")
	err := qs.Filter("name", name).One(user)
	if err == nil {
		return errors.New("name already exist"), ""
	}

	if name == "admin" {
		user.IsAdmin = true
	}

	curve := elliptic.P256()
	private, err := ecdsa.GenerateKey(curve, rand.Reader)
	pubKey := append(private.PublicKey.X.Bytes(), private.PublicKey.Y.Bytes()...)
	user.VHash = hex.EncodeToString(pubKey)

	uuid := new(UUID)
	rand.Read(uuid.Bytes())
	user.UUID = uuid.String()

	_, err = o.Insert(user)
	if err != nil {
		return err, ""
	}
	return nil, user.VHash
}

func GetAllUsers() ([]*User, error) {
	o := orm.NewOrm()

	users := make([]*User, 0)
	qs := o.QueryTable("user")
	_, err := qs.All(&users)
	return users, err
}

func SetExpiredUsers() ([]*User, error) {
	o := orm.NewOrm()

	now := time.Now().UTC()

	users := make([]*User, 0)
	qs := o.QueryTable("user")
	_, err := qs.Filter("expire__gt", check_t).Filter("expire__lt", now).All(&users)
	if err == nil {
		qs.Filter("expire__gt", check_t).Filter("expire__lt", now).Update(orm.Params{"expire": mark_t, "nextrefill": mark_t})
	}
	return users, err
}

func RefillUser(user *User) {
	o := orm.NewOrm()
	user.NextRefill = user.NextRefill.AddDate(0, 1, 0).UTC()
	user.Inbound = 0
	user.Outbound = 0
	o.Update(user)
}

func RefillUsers() ([]*User, error) {
	o := orm.NewOrm()

	now := time.Now().UTC()

	users := make([]*User, 0)
	qs := o.QueryTable("user")
	_, err := qs.Filter("expire__gt", check_t).Filter("nextrefill__lt", now).All(&users)
	return users, err
}

func GetUser(name string) *User {
	o := orm.NewOrm()

	user := new(User)

	qs := o.QueryTable("user")
	err := qs.Filter("name", name).One(user)
	if err == orm.ErrNoRows {
		return nil
	} else {
		return user
	}
}

func GetUserById(id int64) *User {
	o := orm.NewOrm()

	user := new(User)

	qs := o.QueryTable("user")
	err := qs.Filter("id", id).One(user)
	if err == orm.ErrNoRows {
		return nil
	} else {
		return user
	}
}

func VerifyUserEmail(name, hash string) bool {
	o := orm.NewOrm()

	user := new(User)

	qs := o.QueryTable("user")
	err := qs.Filter("name", name).One(user)
	if err == orm.ErrNoRows {
		return false
	}

	if user.VHash == "v" {
		return true
	} else if user.VHash == hash {
		user.VHash = "v"
		o.Update(user)
		return true
	}

	return false
}

func ModifyUserStat(name, inbound, outbound string) (error, bool) {
	o := orm.NewOrm()

	user := new(User)

	qs := o.QueryTable("user")
	err := qs.Filter("name", name).One(user)
	if err == orm.ErrNoRows {
		return err, false
	}

	ib, _ := strconv.ParseFloat(inbound, 64)
	ob, _ := strconv.ParseFloat(outbound, 64)
	if ib == -1 && ob == -1 {
		user.Inbound = 0
		user.Outbound = 0
	} else {
		user.Inbound += ib
		user.Outbound += ob
	}
	o.Update(user)
	return nil, user.Outbound > user.Total
}

func ExpandUserExpire(name string, m int) error {
	o := orm.NewOrm()

	user := new(User)

	qs := o.QueryTable("user")
	err := qs.Filter("name", name).One(user)
	if err == orm.ErrNoRows {
		return err
	}

	tn := time.Now().UTC()
	te := user.Expire
	if tn.After(te) {
		// already expired, start from now
		user.Expire = tn.AddDate(0, m, 0).UTC()
		user.NextRefill = tn.AddDate(0, 1, 0).UTC()
	} else {
		// not expired yet, start from previous expiration
		user.Expire = te.AddDate(0, m, 0).UTC()
		user.NextRefill = te.AddDate(0, 1, 0).UTC()
	}

	o.Update(user)
	return nil
}

func ModifyUserSec(name, pwd string) error {
	o := orm.NewOrm()

	user := new(User)

	qs := o.QueryTable("user")
	err := qs.Filter("name", name).One(user)
	if err == orm.ErrNoRows {
		return err
	}

	user.PWD = pwd
	o.Update(user)
	return nil
}

func ModifyUserKey(name string, val int64) error {
	o := orm.NewOrm()

	user := new(User)

	qs := o.QueryTable("user")
	err := qs.Filter("name", name).One(user)
	if err == orm.ErrNoRows {
		return err
	}

	user.NumKeys += val
	o.Update(user)
	return nil
}

type Host struct {
	Id    int64
	Users int64
	IP    string
}

func GetAllHosts() ([]*Host, error) {
	o := orm.NewOrm()

	hosts := make([]*Host, 0)
	qs := o.QueryTable("host")
	_, err := qs.All(&hosts)
	return hosts, err
}

func RegisterDB() {
	if !com.IsExist(_DB_NAME) {
		os.MkdirAll(path.Dir(_DB_NAME), os.ModePerm)
		os.Create(_DB_NAME)
	}

	orm.RegisterModel(new(Category), new(Topic),
		new(User), new(Comment), new(Host))
	orm.RegisterDriver(_SQLITE3_DRIVER, orm.DRSqlite)
	orm.RegisterDataBase("default", _SQLITE3_DRIVER, _DB_NAME, 10)
}
