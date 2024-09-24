package main

import (
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	"github.com/xorm-io/xorm"
)

const (
	myDsn               = "root:123456@tcp(localhost:3306)/account?charset=utf8"
	casdoorOrganization = "scutbot"
)

func main() {
	myDb, err := xorm.NewEngine("mysql", myDsn)
	if err != nil {
		panic(err)
	}

	migrateAccounts(myDb)
	dumpAccounts(myDb)
}
