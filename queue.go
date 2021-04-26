package main

import (
	"fmt"
	"sync"
	"time"
)

type Queue struct {
	que    []interface{}
	accept chan interface{}
	notify chan interface{}
	cond   sync.Cond
}

func newQueue() *Queue {

	queue := new(Queue)
	queue.que = make([]interface{}, 0)
	queue.cond = *sync.NewCond(&sync.Mutex{})

	queue.accept = make(chan interface{})
	queue.notify = make(chan interface{})
	queue.run()
	return queue
}

func (q *Queue) run() {

	go func() {
		for {
			select {
			case val := <-q.accept:
				q.put(val)
			}
		}
	}()
	go func() {
		for {
			v := q.get()
			q.notify <- v
		}
	}()
}

func (q *Queue) In() chan<- interface{} {
	return q.accept
}

func (q *Queue) Out() <-chan interface{} {
	return q.notify
}

func (q *Queue) put(val interface{}) {

	q.cond.L.Lock()
	q.que = append(q.que, val)
	q.cond.L.Unlock()
	q.cond.Signal()
}

func (q *Queue) get() interface{} {

	q.cond.L.Lock()
	if len(q.que) == 0 {
		q.cond.Wait()
	}
	val := q.que[0]
	q.que = q.que[1:]
	q.cond.L.Unlock()
	return val
}

func _main() {

	queue := newQueue()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		for i := 0; i < 100; i++ {
			time.Sleep(1 * time.Second)
			queue.In() <- i
			fmt.Println("...In ", i)
		}
		wg.Done()
	}()

	go func() {
		// for i := 0; i < 100; i++ {
		// 	j := queue.get()
		// 	fmt.Println("...get ", j)
		// }
		for {
			select {
			case v := <-queue.Out():
				fmt.Println("...Out", v)
			}
			time.Sleep(2 * time.Second)
		}
		wg.Done()
	}()

	wg.Wait()

}
