package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path[1:]
		host, ok := mapping[path]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		web3Req := Web3ClientRequest{}
		body, _ := io.ReadAll(r.Body)
		defer r.Body.Close()

		err := json.Unmarshal(body, &web3Req)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		if len(web3Req.Method) <= 4 || web3Req.Method[0:4] != "mev_" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		target, _ := url.Parse(host)
		proxy := httputil.NewSingleHostReverseProxy(target)
		proxy.Director = func(req *http.Request) {
			req.URL = target
			req.Host = target.Host
			req.Body = io.NopCloser(bytes.NewReader(body))
		}

		proxy.ServeHTTP(w, r)
	})
	panic(http.ListenAndServe(":8080", nil))
}

func init() {
	loadConfig("config.json")
}
