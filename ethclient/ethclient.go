package ethclient

import (
	"bytes"
	"io"
	"net/http"
	"service_agent/config"
	"time"
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

func SendRawTransaction(data []byte) ([]byte, error) {
	resp, err := httpClient.Post(config.GlobalConfig.Sentry, "application/json", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}
