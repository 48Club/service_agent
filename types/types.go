package types

import (
	"encoding/json"
	"errors"
	"log"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	types2 "github.com/ethereum/go-ethereum/core/types"
	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

type Web3ClientRequest struct {
	JsonRPC string        `json:"jsonrpc"`
	Id      interface{}   `json:"id"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
}

type Web3ClientRequests []Web3ClientRequest

type LimitResponse struct {
	Limit     HeaderStrs
	Remaining HeaderStrs
}

type HeaderStrs []string

func (s HeaderStrs) ToString() string {
	b, err := json.Marshal(s)
	if err != nil {
		return "[]"
	}
	return string(b)
}

type QpsStats struct {
	ByMinute map[time.Time]countHelper
	sync.Mutex
}

type countHelper struct {
	Count            int
	EthCallCount     int
	TransactionCount int
}

func NewQpsStats() *QpsStats {
	return &QpsStats{
		ByMinute: make(map[time.Time]countHelper),
	}
}

func (q *QpsStats) Add(c, e, t int) {
	tt := time.Now().Truncate(time.Minute)

	q.Lock()
	defer q.Unlock()

	q.prune(tt)

	qc, ok := q.ByMinute[tt]
	if !ok {
		q.ByMinute[tt] = countHelper{
			Count:            c,
			EthCallCount:     e,
			TransactionCount: t,
		}
		return
	}
	q.ByMinute[tt] = countHelper{
		Count:            qc.Count + c,
		EthCallCount:     qc.EthCallCount + e,
		TransactionCount: qc.TransactionCount + t,
	}
}

func (q *QpsStats) prune(t time.Time) {
	if len(q.ByMinute) < 2 {
		return
	}

	for k, v := range q.ByMinute {
		if k != t {
			log.Printf("[QPS Stat], Time: %s, QPS: %.2f, eth_call QPS: %.2f, eth_sendTransaction QPS: %.2f\n", t.Add(-1*time.Minute), float64(v.Count)/60, float64(v.EthCallCount)/60, float64(v.TransactionCount)/60)
			delete(q.ByMinute, k)
		}
	}
}

type DbTx struct {
	ID        uint64 `gorm:"bigint;primaryKey;autoIncrement;column:id" json:"id"` // pk
	HostName  string `gorm:"varchar(255);column:host_name" json:"-"`              // 主机名 不唯一
	Hash      string `gorm:"varchar(66);column:hash;unique" json:"hash"`          // 交易hash 唯一
	Sender    string `gorm:"varchar(42);column:sender" json:"sender"`             // 发送者 不唯一
	To        string `gorm:"varchar(42);column:to" json:"to"`                     // 接收者 不唯一
	Value     string `gorm:"varchar(66);column:value" json:"value"`               // 交易主网币数量
	Nonce     uint64 `gorm:"bigint;column:nonce" json:"nonce"`                    // 交易nonce
	CreatedAt int64  `gorm:"bigint;column:created" json:"created"`                // 创建时间
}

func (DbTx) TableName() string {
	return "agent_txs"
}

type DbTxs []DbTx

func (txs DbTxs) Create(db *gorm.DB, removeCache func(string)) {
	_ = db.Transaction(func(session *gorm.DB) error {
		for _, tx := range txs {
			result := session.Create(&tx)
			if err := result.Error; err != nil {
				if errors.Is(err, gorm.ErrDuplicatedKey) {
					continue
				}
				if mysqlErr, ok := err.(*mysql.MySQLError); ok && mysqlErr.Number == 1062 {
					continue
				}
				log.Printf("tx insert error: %v", err)
				removeCache(tx.Hash)
			}
		}
		return nil
	})
}

func (txs *DbTxs) Find(db *gorm.DB, host string) error {
	tt := time.Now().Add(-time.Hour * 24 * 7).Unix()
	tx := db.Where("host_name = ? AND created > ?", host, tt).Find(txs)
	return tx.Error
}

type Transactions []*types2.Transaction

func (txs Transactions) TxFormat2DB(h string, isTxExist func(string) bool, addCache func(string)) DbTxs {
	dbTxs := make([]DbTx, 0, len(txs))
	for _, tx := range txs {
		txHash := tx.Hash().Hex()
		if isTxExist(txHash) {
			continue
		}
		to := common.Address{}.Hex()
		if tx.To() != nil {
			to = tx.To().Hex()
		}
		dbTxs = append(dbTxs, DbTx{
			HostName:  h,
			Hash:      txHash,
			Sender:    getTxSender(tx).Hex(),
			To:        to,
			Value:     hexutil.EncodeBig(tx.Value()),
			Nonce:     tx.Nonce(),
			CreatedAt: time.Now().Unix(),
		})
		addCache(txHash)
	}
	return dbTxs
}

func getTxSender(tx *types2.Transaction) (a common.Address) {
	chinaID := tx.ChainId()
	if chinaID == nil {
		return
	}
	a, _ = types2.Sender(types2.NewLondonSigner(chinaID), tx)
	return
}
