package test

import (
	"math/big"
	"testing"

	"github.com/48Club/service_agent/handler"
	"github.com/48Club/service_agent/types"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/assert"
)

func TestHexFormat(t *testing.T) {
	t.Log(hexutil.EncodeBig(big.NewInt(1e9)))
}

func TestMapSetTest(t *testing.T) {
	m := mapset.NewSet[string]()
	m.Append([]string{}...)
	assert.Equal(t, 0, m.Cardinality())
	m.Append("1", "2", "3")
	assert.Equal(t, 3, m.Cardinality())
}

func TestLimitRes(t *testing.T) {
	l := types.LimitResponse{}
	handler.LimitMiddleware2("0.0.0.0", false, 1, &l)
	t.Log(l.Limit.ToString(), l.Remaining.ToString())
}
