package main

import (

	"fmt"
	//"sync"
	"sync"
	"time"

)

type Queue struct {
	que []interface{}
	cond sync.Cond

}

func (queue *Queue) put(val interface{}) {

	queue.cond.L.Lock()
	queue.que = append(queue.que, val)
	queue.cond.L.Unlock()
	queue.cond.Signal()

}

func (queue *Queue) get() interface{} {

	queue.cond.L.Lock()
	if len(queue.que) == 0 {
		queue.cond.Wait()
	}
	fmt.Println("wait end")
	val := queue.que[0]
	queue.que = queue.que[1:]
	queue.cond.L.Unlock()
	return val
}
	

func main() {

	queue := new(Queue)
	queue.que = make([]interface{}, 0)
	queue.cond = *sync.NewCond(&sync.Mutex{})

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {

		for i:=0;i<100;i++ {
			time.Sleep(2*time.Second)
			queue.put(i)
			fmt.Println("...put ", i)			
		}
		wg.Done()
	}()

	go func() {
		for i:=0;i<100;i++ {
			j := queue.get()
			fmt.Println("...get ", j)
		}
		wg.Done()
	}()

	wg.Wait()
		
		
}	
