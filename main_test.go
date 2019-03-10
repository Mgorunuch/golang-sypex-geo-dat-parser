package main

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"os"
	"testing"
)

type TestItem struct {
	Ip      string `json:"ip"`
	Country string `json:"country"`
}

var testItems []TestItem

func TestCountryDetection(t *testing.T) {
	jsonItems, err := ioutil.ReadFile("./example.json")
	if err != nil {
		log.Print(err)
	}

	err = json.Unmarshal(jsonItems, &testItems)
	if err != nil {
		log.Println("error:", err)
	}

	fileStat, _ := os.Stat("./SxGeo.dat")
	r, err := os.Open("./SxGeo.dat")
	if err != nil {
		log.Panic(err)
	}

	defer r.Close()

	fileSize := fileStat.Size()

	data := make([]byte, fileSize)
	io.ReadFull(r, data[:])

	geo := NewSxGeo(data)

	for _, item := range testItems {
		answer := geo.GetCountryByIp(item.Ip)
		log.Print(answer == item.Country)
	}
	log.Print(geo.GetCountryByIp("188.163.89.66"))
}
