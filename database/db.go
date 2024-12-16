package database

import (
	"sync"
	"time"

	"github.com/48Club/service_agent/config"
	"github.com/48Club/service_agent/types"
	"github.com/ethereum/go-ethereum/common/lru"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type server struct {
	txHashCache *lru.Cache[string, struct{}]
	*gorm.DB
	rw sync.RWMutex
}

var (
	Server = server{}
)

// create database agent
func init() {
	engine, err := gorm.Open(mysql.Open(config.GlobalConfig.DSN()), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent), // 关闭 MySQL 日志
	})
	if err != nil {
		panic(err)
	}

	err = engine.AutoMigrate(&types.DbTxs{})
	if err != nil {
		panic(err)
	}

	Server.DB = engine
	Server.txHashCache = lru.NewCache[string, struct{}](5000)

	go loadCache()

	go pruneDB()
}

func loadCache() {
	var hashs []string
	Server.Model(&types.DbTx{}).Limit(5000).Pluck("hash", &hashs)
	for _, hash := range hashs {
		Server.txHashCache.Add(hash, struct{}{})
	}
}

func pruneDB() {
	for {
		tx := Server.Unscoped().Where("created < ?", time.Now().Add(-time.Hour*24*7).Unix()).Limit(1000).Delete(&types.DbTx{}).Commit()
		if tx.RowsAffected < 1000 {
			time.Sleep(time.Hour)
		}
		time.Sleep(time.Second)
	}
}
func IsTxExist(hash string) bool {
	Server.rw.RLock()
	defer Server.rw.RUnlock()
	_, ok := Server.txHashCache.Get(hash)
	return ok
}

func AddCache(hash string) {
	Server.rw.Lock()
	defer Server.rw.Unlock()
	Server.txHashCache.Add(hash, struct{}{})
}

func RemoveCache(hash string) {
	Server.rw.Lock()
	defer Server.rw.Unlock()
	Server.txHashCache.Remove(hash)
}

func WalletCount(db *gorm.DB, host string) (count int64) {
	limit7days(db.Model(&types.DbTx{})).Where("host_name = ?", host).Group("sender").Count(&count)
	return
}

func limit7days(tx *gorm.DB) *gorm.DB {
	tt := time.Now().Add(-time.Hour * 24 * 7).Unix()
	return tx.Where("created > ?", tt)
}

func TxCount(db *gorm.DB, host string) (count int64) {
	limit7days(db.Model(&types.DbTx{})).Where("host_name = ?", host).Count(&count)
	return
}

func WalletList(db *gorm.DB, host string) (wallets []string) {
	limit7days(db.Model(&types.DbTx{})).Where("host_name = ?", host).Group("sender").Pluck("sender", &wallets)
	return
}
