package main

import (
	"log"
	"os"
)

func main() {
	path := "/dev/hidg1"

	fd, err := os.OpenFile(path, os.O_WRONLY, 0644)
	if err != nil {
		log.Println("hid open error", err)
		return
	}

	log.Println("hid open")

	err = fd.Close()
	if err != nil {
		log.Println("hid close error", err)
	}

	log.Println("hid close")
}

// CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 go build -o output/hid_test tests/hid.go
