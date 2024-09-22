// Copyright 2021 The Casdoor Authors. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"fmt"
	"github.com/beego/beego"
	"github.com/beego/beego/logs"
	_ "github.com/beego/beego/session/redis"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/adaptor"
	"github.com/gofiber/fiber/v3/middleware/recover"
	jsoniter "github.com/json-iterator/go"
	"github.com/scutrobotlab/casdoor/authz"
	"github.com/scutrobotlab/casdoor/conf"
	"github.com/scutrobotlab/casdoor/ldap"
	"github.com/scutrobotlab/casdoor/object"
	"github.com/scutrobotlab/casdoor/proxy"
	"github.com/scutrobotlab/casdoor/radius"
	"github.com/scutrobotlab/casdoor/routers"
	"github.com/scutrobotlab/casdoor/util"
	"time"
)

func main() {
	object.InitFlag()
	object.InitAdapter()
	object.CreateTables()

	object.InitDb()
	object.InitDefaultStorageProvider()
	object.InitLdapAutoSynchronizer()
	proxy.InitHttpClient()
	authz.InitApi()
	object.InitUserManager()
	object.InitFromFile()
	object.InitCasvisorConfig()

	util.SafeGoroutine(func() { object.RunSyncUsersJob() })

	// beego.DelStaticPath("/static")
	// beego.SetStaticPath("/static", "web/build/static")

	beego.BConfig.WebConfig.DirectoryIndex = true
	beego.SetStaticPath("/swagger", "swagger")
	beego.SetStaticPath("/files", "files")
	// https://studygolang.com/articles/2303
	beego.InsertFilter("*", beego.BeforeRouter, routers.StaticFilter)
	beego.InsertFilter("*", beego.BeforeRouter, routers.AutoSigninFilter)
	beego.InsertFilter("*", beego.BeforeRouter, routers.CorsFilter)
	beego.InsertFilter("*", beego.BeforeRouter, routers.ApiFilter)
	beego.InsertFilter("*", beego.BeforeRouter, routers.PrometheusFilter)
	beego.InsertFilter("*", beego.BeforeRouter, routers.RecordMessage)
	beego.InsertFilter("*", beego.AfterExec, routers.AfterRecordMessage, false)

	beego.BConfig.WebConfig.Session.SessionOn = true
	beego.BConfig.WebConfig.Session.SessionName = "casdoor_session_id"
	if conf.GetConfigString("redisEndpoint") == "" {
		beego.BConfig.WebConfig.Session.SessionProvider = "file"
		beego.BConfig.WebConfig.Session.SessionProviderConfig = "./tmp"
	} else {
		beego.BConfig.WebConfig.Session.SessionProvider = "redis"
		beego.BConfig.WebConfig.Session.SessionProviderConfig = conf.GetConfigString("redisEndpoint")
	}
	beego.BConfig.WebConfig.Session.SessionCookieLifeTime = 3600 * 24 * 30
	// beego.BConfig.WebConfig.Session.SessionCookieSameSite = http.SameSiteNoneMode

	err := logs.SetLogger(logs.AdapterFile, conf.GetConfigString("logConfig"))
	if err != nil {
		panic(err)
	}
	port := beego.AppConfig.DefaultInt("httpport", 8000)
	// logs.SetLevel(logs.LevelInformational)
	logs.SetLogFuncCall(false)

	go ldap.StartLdapServer()
	go radius.StartRadiusServer()
	go object.ClearThroughputPerSecond()

	go beego.Run(":0")
	time.Sleep(1 * time.Second)
	beego.BeeApp.Server.Shutdown(context.Background())

	app := fiber.New(fiber.Config{
		JSONEncoder: jsoniter.Marshal,
		JSONDecoder: jsoniter.Unmarshal,
	})

	app.Use(recover.New())

	app.All("/*", adaptor.HTTPHandler(beego.BeeApp.Server.Handler))

	app.Listen(fmt.Sprintf(":%d", port))
}
