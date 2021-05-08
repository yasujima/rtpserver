package main

import (
	"log"
)

func f(mainc chan chan interface{}) {

	c := make(chan interface{})

	mainc <- c
	go func() {
		c <- string("hello")
		select {
		case v := <-c:
			log.Print("func call back ", v.(string))
		}
	} ()
}

func main() {


	mainc := make(chan chan interface{})
	go func() {
		f(mainc)
	}()
	select {
	case c := <-mainc:
		msg := <-c
		c <- "---" + msg.(string) + "---"
	}
}

	
