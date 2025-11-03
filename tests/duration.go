package main

import (
	"log"
	"time"
)

func main() {
	t1 := time.UnixMicro(1735105983)
	t2 := time.UnixMicro(1735122649)

	log.Println("duration", t2.Sub(t1), t1.Sub(t2))
}
