package limit

import (
	"fmt"
	"testing"
	"time"
)

func TestLimit(t *testing.T) {
	limit := 500000
	window := 7 * time.Second
	rl := NewIPBasedRateLimiter(limit, window, "")
	allowedCount, deniedCount := 0, 0
	for i := 0; i < 1000000; i++ {
		if rl.Allow("8.8.8.8", false, 1).Allow {
			allowedCount++
		} else {
			deniedCount++
		}
	}
	fmt.Printf("Allowed: %d, Denied: %d\n", allowedCount, deniedCount)
}
