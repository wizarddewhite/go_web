package models

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strconv"
	"time"

	"github.com/astaxie/beego/orm"
	_ "github.com/mattn/go-sqlite3"
)

var mark_t = time.Date(2009, 11, 17, 20, 34, 58, 0, time.UTC)
var check_t = time.Date(2009, 11, 17, 20, 34, 59, 0, time.UTC)
var byteGroups = []int{8, 4, 4, 4, 12}

type User struct {
	Id      int64
	Name    string `orm:"index"`
	Email   string
	VHash   string
	Reset   string
	IsAdmin bool
	PWD     string
	UUID    string

	Recommend  string
	Recommends int

	Phone   string
	Passwd  string
	BHId    string
	BHToken string

	// expire
	Expire     time.Time
	NextRefill time.Time
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

func AddUser(name, email, pwd, recommend string) (error, string, string) {
	o := orm.NewOrm()

	user := &User{
		Name:       name,
		PWD:        pwd,
		Email:      email,
		Expire:     mark_t,
		NextRefill: mark_t,
		Recommend:  recommend,
		Recommends: 0,
	}

	qs := o.QueryTable("user")
	err := qs.Filter("name", name).One(user)
	if err == nil {
		return errors.New("name already exist"), "", ""
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
		return err, "", ""
	}
	return nil, user.VHash, user.UUID
}

func TotalUsers() int64 {
	o := orm.NewOrm()

	users := make([]*User, 0)
	qs := o.QueryTable("user")
	qs.All(&users)
	count, _ := qs.Count()
	return count
}

func GetAllUsers(limit int64, offset int64) ([]*User, int64, error) {
	o := orm.NewOrm()

	users := make([]*User, 0)
	qs := o.QueryTable("user")
	_, err := qs.Limit(limit, offset).All(&users)
	count, _ := qs.Count()
	return users, count, err
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

// Users whose Expire is greater than check_t and less than now are expired.
// Set their Expire to mark_t and return them.
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

// Expand NextRefill for one month
func RefillUser(user *User) {
	o := orm.NewOrm()
	user.NextRefill = user.NextRefill.AddDate(0, 1, 0).UTC()
	o.Update(user)
}

// Users whose Expire is greater than check_t and NextRefill is less than now
// need refile.
func RefillUsers() ([]*User, error) {
	o := orm.NewOrm()

	now := time.Now().UTC()

	users := make([]*User, 0)
	qs := o.QueryTable("user")
	_, err := qs.Filter("expire__gt", check_t).Filter("nextrefill__lt", now).All(&users)
	return users, err
}

func GetUserRecommend(name string) (recommend string) {
	o := orm.NewOrm()

	user := new(User)

	qs := o.QueryTable("user")
	err := qs.Filter("name", name).One(user)
	if err == orm.ErrNoRows {
		return ""
	}

	recommend = user.Recommend
	user.Recommend = ""

	o.Update(user)
	return
}

func IncUserRecommend(name string) error {
	o := orm.NewOrm()

	user := new(User)

	qs := o.QueryTable("user")
	err := qs.Filter("name", name).One(user)
	if err == orm.ErrNoRows {
		return err
	}

	user.Recommends += 1

	o.Update(user)
	return nil
}

// Expand User Expire for m month
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
	}

	o.Update(user)
	return nil
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
		/* place holder */
	} else {
		/* place holder */
	}
	o.Update(user)
	return nil, true
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

func ModifyUserPS(name, pn, eps, id, token string) error {
	o := orm.NewOrm()

	user := new(User)

	qs := o.QueryTable("user")
	err := qs.Filter("name", name).One(user)
	if err == orm.ErrNoRows {
		return err
	}

	user.Phone = pn
	user.Passwd = eps
	user.BHId = id
	user.BHToken = token
	o.Update(user)
	return nil
}

func VerifyUserEmail(name, hash string) error {
	o := orm.NewOrm()

	user := new(User)

	qs := o.QueryTable("user")
	err := qs.Filter("name", name).One(user)
	if err == orm.ErrNoRows {
		return errors.New("Email not confirmed!")
	}

	if user.VHash == "v" {
		return errors.New("Email already confirmed!")
	} else if user.VHash == hash {
		user.VHash = "v"
		o.Update(user)
		return nil
	}

	return errors.New("Email not confirmed!")
}

func GetUserResetHash(name string) (string, string) {
	o := orm.NewOrm()

	user := new(User)

	qs := o.QueryTable("user")
	err := qs.Filter("name", name).One(user)
	if err == orm.ErrNoRows {
		return "", ""
	}

	curve := elliptic.P256()
	private, err := ecdsa.GenerateKey(curve, rand.Reader)
	pubKey := append(private.PublicKey.X.Bytes(), private.PublicKey.Y.Bytes()...)
	user.Reset = hex.EncodeToString(pubKey)
	o.Update(user)
	return user.Email, user.Reset
}

func ResetUser(name, hash, pwd string) error {
	o := orm.NewOrm()

	user := new(User)

	qs := o.QueryTable("user")
	err := qs.Filter("name", name).One(user)
	if err == orm.ErrNoRows {
		return errors.New("No such user!")
	}

	if user.Reset == hash {
		user.PWD = pwd
		user.Reset = "v"
		o.Update(user)
		return nil
	}

	return errors.New("Reset link is not valid!")
}
