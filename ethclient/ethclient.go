package ethclient

import (
	"bytes"
	"io"
	"net/http"
	"time"

	"github.com/48Club/service_agent/config"
)

var httpClient *http.Client

func init() {
	hc := http.DefaultClient
	hc.Timeout = time.Second * 60
	hc.Transport = &http.Transport{
		MaxIdleConns:        2<<15 - 1,
		MaxIdleConnsPerHost: 2<<15 - 1,
	}

	httpClient = hc
}

func Send2Sentry(data []byte) ([]byte, error) {
	resp, err := httpClient.Post(config.GlobalConfig.Sentry, "application/json", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}
