package test

import (
	"strings"
	"testing"
	"time"
)

func TestByteSpace(t *testing.T) {
	var bs [][]byte
	bs = append(bs, []byte("   {}"))
	bs = append(bs, []byte("   [{}]"))
	count := 2 << 28
	begin := time.Now()
	for i := 0; i < count; i++ {
		for _, b := range bs {
			_ = strings.Trim(string(b), " ")[0]
			/*
				Running tool: /usr/local/go/bin/go test -timeout 30s -run ^TestByteSpace$ service_agent/test

				=== RUN   TestByteSpace
				    /Users/lifa/Documents/GitHub/service_agent/test/byte_test.go:48: 536870912 7.558964792s
				--- PASS: TestByteSpace (7.56s)
				PASS
				ok      service_agent/test      8.586s
			*/
			for len(b) > 0 && b[0] == 32 {
				/*
					Running tool: /usr/local/go/bin/go test -timeout 30s -run ^TestByteSpace$ service_agent/test

					=== RUN   TestByteSpace
					    /Users/lifa/Documents/GitHub/service_agent/test/byte_test.go:55: 536870912 3.035139708s
					--- PASS: TestByteSpace (3.04s)
					PASS
					ok      service_agent/test      3.458s
				*/
				b = b[1:]
			}
			for _, v := range b {
				/*
					Running tool: /usr/local/go/bin/go test -timeout 30s -run ^TestByteSpace$ service_agent/test

					=== RUN   TestByteSpace
					    /Users/lifa/Documents/GitHub/service_agent/test/byte_test.go:55: 536870912 3.043875917s
					--- PASS: TestByteSpace (3.04s)
					PASS
					ok      service_agent/test      3.401s
				*/
				if v != 32 {
					break
				}
			}
		}
	}
	end := time.Now()
	t.Log(count, end.Sub(begin))
}
