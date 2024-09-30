package types

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCheckCallGas(t *testing.T) {
	str := `
{
    "id": 1,
    "jsonrpc": "2.0",
    "method": "eth_call",
    "params": [
        {
            "gas": "0xb"
        },
        {
            "gas": "0xa"
        },
        "latest"
    ]
}
  `
	req := Web3ClientRequest{}
	err := json.Unmarshal([]byte(str), &req)
	assert.NoError(t, err)
	assert.Equal(t, uint64(21), req.CheckCallGas())
}
