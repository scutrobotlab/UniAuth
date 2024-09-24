package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/scutrobotlab/casdoor/object"
	"github.com/scutrobotlab/casdoor/util"

	"github.com/beego/beego/logs"
	"github.com/xorm-io/xorm"
)

const myAvatarUrl = "https://my.scutbot.cn/storage/avatars/users/%s.jpg"

//go:embed cookie
var requestCookie []byte

type MyUser struct {
	Id        int64      `xorm:"pk autoincr" json:"id"`
	Uuid      string     `xorm:"varchar(100) notnull unique" json:"uuid"`
	Avatar    string     `xorm:"varchar(100)" json:"avatar"`
	Name      string     `xorm:"varchar(100) notnull" json:"name"`
	Email     string     `xorm:"varchar(100) notnull unique" json:"email"`
	Password  string     `xorm:"varchar(100) notnull" json:"password"`
	ApiToken  string     `xorm:"varchar(100)" json:"-"`
	DeletedAt *time.Time `xorm:"deleted_at" json:"-"`
	CreatedAt time.Time  `xorm:"created" json:"-"`
	UpdatedAt time.Time  `xorm:"updated" json:"-"`

	Season int `xorm:"-" json:"season"`
}

type MyGroup struct {
	Id        int64      `xorm:"pk autoincr"`
	Name      string     `xorm:"varchar(100) notnull"`
	DeletedAt *time.Time `xorm:"deleted_at"`
	CreatedAt time.Time  `xorm:"created"`
	UpdatedAt time.Time  `xorm:"updated"`
}

type MyPower struct {
	UserId    int64     `xorm:"pk"`
	GroupId   int64     `xorm:"pk"`
	Power     int       `xorm:"notnull"`
	Status    int       `xorm:"notnull"`
	Token     string    `xorm:"varchar(100)"`
	CreatedAt time.Time `xorm:"created"`
	UpdatedAt time.Time `xorm:"updated"`
}

func migrateAccounts(myDb *xorm.Engine) {
	users := make([]MyUser, 0)
	err := myDb.Table("users").Find(&users)
	if err != nil {
		logs.Error(err)
		return
	}

	var wg sync.WaitGroup
	wg.Add(len(users))
	for _, user := range users {
		go func(user MyUser) {
			defer wg.Done()
			migrateAccount(myDb, &user)
		}(user)
	}

	wg.Wait()
}

func dumpAccounts(myDb *xorm.Engine) {
	users := make([]MyUser, 0)
	err := myDb.Table("users").Find(&users)
	if err != nil {
		logs.Error(err)
		return
	}

	var wg sync.WaitGroup
	wg.Add(len(users))
	for _, user := range users {
		go func(user MyUser) {
			defer wg.Done()
			dumpAccount(myDb, &user)
		}(user)
	}

	wg.Wait()
}

func dumpAccount(myDb *xorm.Engine, user *MyUser) {
	if user.DeletedAt != nil {
		return
	}

	dir := fmt.Sprintf("./migrate/account/%d", user.Id)
	err := os.MkdirAll(dir, 0o755)
	if err != nil {
		logs.Error("Failed to create directory %s", dir)
		return
	}

	dumpUserInfo(user, dir)
	dumpUserAvatar(user, dir)
	dumpUserPower(myDb, user, dir)

	logs.Info("Dumped user %d", user.Id)
}

func dumpUserInfo(user *MyUser, dir string) {
	infoFile, err := os.OpenFile(fmt.Sprintf("%s/info.json", dir), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
	}

	user.Season = user.CreatedAt.Year() + 1

	jsonContent, err := json.MarshalIndent(user, "", "    ")
	if err != nil {
		logs.Error("Failed to marshal user %d", user.Id)
		return
	}

	_, err = infoFile.Write(jsonContent)
	if err != nil {
		logs.Error("Failed to write user %d", user.Id)
		return
	}
}

func dumpUserAvatar(user *MyUser, dir string) {
	avatarUrl := fmt.Sprintf(myAvatarUrl, user.Uuid)
	avatarFile, err := os.OpenFile(fmt.Sprintf("%s/avatar.jpg", dir), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		logs.Error("Failed to create avatar file for user %d", user.Id)
		return
	}

	req, err := http.NewRequest("GET", avatarUrl, nil)
	if err != nil {
		logs.Error("Failed to create request for user %d", user.Id)
		return
	}

	req.Header.Set("Cookie", string(requestCookie))

	avatar, err := http.DefaultClient.Do(req)
	if err != nil {
		logs.Error("Failed to download avatar for user %d", user.Id)
		return
	}

	defer avatar.Body.Close()

	if avatar.StatusCode != 200 {
		logs.Error("Failed to download avatar for user %d: %d", user.Id, avatar.StatusCode)
		return
	}

	if avatar.Header.Get("Content-Type") != "image/jpeg" {
		logs.Error("Failed to download avatar for user %d: %s", user.Id, avatar.Header.Get("Content-Type"))
		return
	}

	if avatar.ContentLength < 128 {
		logs.Error("Failed to download avatar for user %d: %d", user.Id, avatar.ContentLength)
		return
	}

	_, err = io.Copy(avatarFile, avatar.Body)
	if err != nil {
		logs.Error("Failed to save avatar for user %d", user.Id)
	}
}

type Power struct {
	Name  string
	Power int
}

func (p *Power) PowerName() string {
	suffix := "组员"
	if p.Power == 2 {
		suffix = "组长"
	}

	return fmt.Sprintf("%s:%s", p.Name, suffix)
}

func dumpUserPower(myDb *xorm.Engine, user *MyUser, dir string) {
	powers := make([]Power, 0)
	err := myDb.Table("powers").
		Join("INNER", "groups", "powers.group_id = groups.id").
		Where("powers.user_id = ?", user.Id).
		Where("powers.status = 2").
		Where("groups.deleted_at IS NULL").
		Where("powers.token IS NULL").
		Find(&powers)
	if err != nil {
		logs.Error("Failed to query powers for user %d", user.Id)
		return
	}

	powerFile, err := os.OpenFile(fmt.Sprintf("%s/power.json", dir), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		logs.Error("Failed to create power file for user %d", user.Id)
		return
	}

	powerNames := make([]string, 0)
	for _, power := range powers {
		powerNames = append(powerNames, power.PowerName())
	}

	jsonContent, err := json.MarshalIndent(powerNames, "", "    ")
	if err != nil {
		logs.Error("Failed to marshal powers for user %d", user.Id)
		return
	}

	_, err = powerFile.Write(jsonContent)
	if err != nil {
		logs.Error("Failed to write powers for user %d", user.Id)
		return
	}
}

func migrateAccount(myDb *xorm.Engine, user *MyUser) {
	if user.DeletedAt != nil {
		return
	}

	casdoorUser := object.User{
		Owner:        casdoorOrganization,
		DisplayName:  user.Name,
		Id:           user.Uuid,
		Email:        user.Email,
		Password:     user.Password,
		PasswordType: "bcrypt",
		CountryCode:  "CN",
		Name:         fmt.Sprintf("user-%s", util.GetRandomName()),
	}

	err := casdoorUser.UpdateUserHash()
	if err != nil {
		logs.Error("Failed to update user hash for user %d: %v", user.Id, err)
		return
	}

	ok, err := object.AddUser(&casdoorUser)
	if err != nil {
		logs.Error("Failed to add user for user %d: %v", user.Id, err)
		return
	}

	if !ok {
		logs.Error("Failed to add user for user %d", user.Id)
		return
	}
}
