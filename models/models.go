package models

import (
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
	Id       int64
	Name     string `orm:"index"`
	IsAdmin  bool
	PWD      string
	Total    int64
	Inbound  float64
	Outbound float64
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

func AddUser(name, pwd string) error {
	o := orm.NewOrm()

	user := &User{
		Name: name,
		PWD:  pwd,
	}

	qs := o.QueryTable("user")
	err := qs.Filter("name", name).One(user)
	if err == nil {
		return errors.New("name already exist")
	}

	if name == "weiyang" {
		user.IsAdmin = true
	}

	_, err = o.Insert(user)
	if err != nil {
		return err
	}
	return nil
}

func GetAllUsers() ([]*User, error) {
	o := orm.NewOrm()

	users := make([]*User, 0)
	qs := o.QueryTable("user")
	_, err := qs.All(&users)
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

func ModifyUserStat(name, inbound, outbound string) error {
	o := orm.NewOrm()

	user := new(User)

	qs := o.QueryTable("user")
	err := qs.Filter("name", name).One(user)
	if err == orm.ErrNoRows {
		return err
	}

	var val float64
	val, _ = strconv.ParseFloat(inbound, 64)
	user.Inbound += val
	val, _ = strconv.ParseFloat(outbound, 64)
	user.Outbound += val
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

func RegisterDB() {
	if !com.IsExist(_DB_NAME) {
		os.MkdirAll(path.Dir(_DB_NAME), os.ModePerm)
		os.Create(_DB_NAME)
	}

	orm.RegisterModel(new(Category), new(Topic), new(User), new(Comment))
	orm.RegisterDriver(_SQLITE3_DRIVER, orm.DRSqlite)
	orm.RegisterDataBase("default", _SQLITE3_DRIVER, _DB_NAME, 10)
}
