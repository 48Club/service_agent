package database

import (
	"github.com/48Club/service_agent/config"
	"github.com/48Club/service_agent/types"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type server struct {
	*gorm.DB
}

var Server = server{}

// create database agent
func init() {
	engine, err := gorm.Open(mysql.Open(config.GlobalConfig.DSN()), &gorm.Config{})
	if err != nil {
		panic(err)
	}

	err = engine.AutoMigrate(&types.DbTxs{})
	if err != nil {
		panic(err)
	}
	Server.DB = engine
}

func WalletCount(db *gorm.DB, host string) (count int64) {
	db.Model(&types.DbTx{}).Where("host_name = ?", host).Group("sender").Count(&count)
	return
}

func TxCount(db *gorm.DB, host string) (count int64) {
	db.Model(&types.DbTx{}).Where("host_name = ?", host).Count(&count)
	return
}

func WalletList(db *gorm.DB, host string) (wallets []string) {
	db.Model(&types.DbTx{}).Where("host_name = ?", host).Group("sender").Pluck("sender", &wallets)
	return
}
