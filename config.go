package main

import (
	"encoding/json"
	"os"
)

var mapping = map[string]string{}

func loadConfig(f string) {
	file, err := os.ReadFile(f)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(file, &mapping)
	if err != nil {
		panic(err)
	}
}
