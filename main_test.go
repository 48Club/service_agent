package main

import "testing"

func TestMaxUint64(t *testing.T) {
	var conn uint64 = ^uint64(0)
	t.Log(conn)
}
