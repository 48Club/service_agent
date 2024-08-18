package test

import (
	"math/big"
	"net"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
)

func TestHexFormat(t *testing.T) {

	t.Log(hexutil.EncodeBig(big.NewInt(1e9)))
}

func TestIPv6Formart(t *testing.T) {
	_, cidr64, _ := net.ParseCIDR("2a00:1390:5:1033:ddf5:ef26:45f5:f7cc/64")
	t.Log(cidr64.String())
}
