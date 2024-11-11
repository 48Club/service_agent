package types

import (
	"encoding/json"
	"log"
	"sync"
	"time"
)

type Web3ClientRequest struct {
	JsonRPC string      `json:"jsonrpc"`
	Id      interface{} `json:"id"`
	Method  string      `json:"method"`
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
