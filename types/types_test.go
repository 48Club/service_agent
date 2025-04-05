package types

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestEncodeBig2HexStr(t *testing.T) {
	assert.Equal(t, "0x63", hexutil.EncodeBig(big.NewInt(99)))
}

func testfunc1(db *gorm.DB, host string) int64 {
	return 123
}
func testfunc2(db *gorm.DB, host string) int64 {
	return 456
}
func testfunc3(db *gorm.DB, host string) []string {
	return []string{"test", "test2"}
}

func TestInterface2Func(t *testing.T) {
	var is []interface{} = []interface{}{testfunc1, testfunc2, testfunc3}
	for index, i := range is {
		switch x := i.(type) {
		case (func(*gorm.DB, string) int64):
			if index == 0 {
				assert.Equal(t, int64(123), x(&gorm.DB{}, "test"))
			} else {
				assert.Equal(t, int64(456), x(&gorm.DB{}, "test"))
			}
		case func(*gorm.DB, string) []string:
			assert.Equal(t, []string{"test", "test2"}, x(&gorm.DB{}, "test"))
		}
	}
}

func TestDecodeRpc(t *testing.T) {
	var body = []byte(`{"jsonrpc":"2.0","method":"eth_getBlockTransactionCountByNumber","params":["0xad1328d13f833b8af722117afdc406a762033321df8e48c00cd372d462f48169",true],"id":1}`)
	var w Web3ClientRequest
	err := json.Unmarshal(body, &w)
	assert.Nil(t, err)
	for index, p := range w.Params {
		s, ok := p.(string)
		if ok {
			t.Logf("[%d]: %s", index, s)
		}
	}
	t.Logf("%+v", w)

}

func TestTx(t *testing.T) {
	var rlpTx rlp.RawValue = hexutil.MustDecode("0x02f8a9388223d88080829f3a9455d398326f99059ff775485246999027b319795580b844a9059cbb000000000000000000000000c37ac5194e1fb34a0935ed42ecb861991755913e0000000000000000000000000000000000000000000000000000000000000000c080a056d20e26818edbe771e1278ecfb1563f758e4ac439ffd7b62149bd2a31435004a04a6df5516d0e20e9471ea4a08bf8c32471ab85ba09134bc1002acb22d1b9036b")
	var tx types.Transaction
	err := tx.UnmarshalBinary(rlpTx)
	assert.Nil(t, err)
	assert.Equal(t, common.HexToAddress("0x55d398326f99059fF775485246999027B3197955"), *tx.To())
	// assert.Equal(t, common.HexToAddress("0xc37Ac5194E1fB34A0935ed42EcB861991755913E"), getTxSender(&tx))
	assert.Equal(t, big.NewInt(0), tx.Value())
}
