package test

import (
	"encoding/json"
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

func TestXxx(t *testing.T) {
	s := `{"jsonrpc":"2.0","method":"eth_call","params":[{"to":"0x0000000000000000000000000000000000000048","value":"0x30"}],"id":83}`
	var w3 types.Web3ClientRequest
	err := json.Unmarshal([]byte(s), &w3)
	assert.Nil(t, err)
	var resp interface{} = 123
	// assert.True(t, decodeEthCall(&resp, w3.Method, false, w3.Params))
	assert.Equal(t, "0x30", resp)
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
	handler.LimitMiddleware2("0.0.0.0", false, 1, &l, "")
	t.Log(l.Limit.ToString(), l.Remaining.ToString())
}
