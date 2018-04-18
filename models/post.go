package models

import (
	"errors"

	"github.com/astaxie/beego/orm"
	_ "github.com/mattn/go-sqlite3"
)

type Post struct {
	Id int64

	UserN string
	ArtId string
	Title string
	Ups   int64
}

func AddPost(usern, artId, title string, ups int64) error {
	o := orm.NewOrm()

	post := &Post{
		UserN: usern,
		ArtId: artId,
		Title: title,
		Ups:   ups,
	}

	qs := o.QueryTable("post")
	err := qs.Filter("artid", artId).One(post)
	if err == nil {
		return errors.New("post already exist")
	}

	_, err = o.Insert(post)
	return err
}

func GetAllPosts(limit int64, offset int64) ([]*Post, int64, error) {
	o := orm.NewOrm()

	posts := make([]*Post, 0)
	qs := o.QueryTable("post")
	_, err := qs.Limit(limit, offset).All(&posts)
	count, _ := qs.Count()
	return posts, count, err
}
