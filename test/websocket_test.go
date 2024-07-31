package test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
)

func TestWsConn(t *testing.T) {
	ec, err := ethclient.Dial("wss://0.48.club/ws/")
	if err != nil {
		t.Fatal(err)
	}
	defer ec.Close()
	var i = 0
	var wg sync.WaitGroup
	for {
		i++
		wg.Add(1)
		go func() {
			defer wg.Done()
			b, err := ec.SuggestGasPrice(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			t.Log(b)
		}()

		if i >= 81 {
			break
		}
	}
	wg.Wait()
	t.Log("done")
}

func TestWsConn2(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		for {
			select {
			case <-ctx.Done():
				t.Log("done")
				return
			default:
				t.Log("break")
				break
			}
		}
	}()

	go func() {
		for {
			select {
			case <-ctx.Done():
				t.Log("done")
				return
			default:
				t.Log("break")
				break
			}
		}
	}()
	time.Sleep(time.Second * 1)
	cancel()
	time.Sleep(time.Second * 1)
}
