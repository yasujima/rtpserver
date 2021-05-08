package main

import (
	//"fmt"
	"time"
	//	"io"
	"log"
)

func createbuf() []byte {
	var buf []byte

	for i := byte(0); i < 0xff; i++ {
		buf = append(buf, i)
	}

	return buf
}

func loop() {
	for i := 0; i < 1000; i++ {
		buf1 := createbuf()
		buf2 := createbuf()
		buf1 = append(buf1, buf2...)
		_ = buf1
	}
}

func loop2() {
	for i := 0; i < 1000; i++ {
		buf1 := createbuf()
		buf2 := createbuf()
		var buf3 = make([]byte, 0, 1000)
		buf3 = append(buf1, buf2...)
		_ = buf3
	}

}

func main() {

	beforeTime := time.Now()

	loop()

	log.Printf("1 time elapsed %s", time.Now().Sub(beforeTime))

	beforeTime = time.Now()

	loop2()

	log.Printf("2 time elapsed %s", time.Now().Sub(beforeTime))
}
