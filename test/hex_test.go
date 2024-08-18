package test

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
)

func TestHexFormat(t *testing.T) {

	t.Log(hexutil.EncodeBig(big.NewInt(1e9)))
}
