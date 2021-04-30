package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

type Queue struct {
	que    []interface{}
	accept chan interface{}
	notify chan interface{}
	cond   sync.Cond
	ctx    context.Context
}

func newQueue(ctx context.Context, recv chan interface{}) *Queue {

	queue := new(Queue)
	queue.que = make([]interface{}, 0)
	queue.cond = *sync.NewCond(&sync.Mutex{})

	//	queue.accept = make(chan interface{})
	queue.accept = recv
	queue.notify = make(chan interface{})
	queue.ctx = ctx
	queue.run()
	return queue
}

func (q *Queue) run() {

	go func() {
		for {
			select {
			case val := <- q.accept:
				q.put(val)
			case <- q.ctx.Done():
				q.put(nil)
				log.Print("queue put canceled")				
				return
			}
		}

	}()
	go func() {
		defer close(q.notify)
		for {
			v := q.get()
			if v != nil {
				q.notify <- v
			}
			select {
			case <- q.ctx.Done():
				log.Print("queue get canceled")
				return
			default:
			}
		}
	}()
}

func (q *Queue) Put() chan<- interface{} {
	return q.accept
}

func (q *Queue) Get() <-chan interface{} {
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

	queue := newQueue(context.Background(), make(chan interface{}))

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		for i := 0; i < 100; i++ {
			time.Sleep(1 * time.Second)
			queue.Put() <- i
			fmt.Println("...Put ", i)
		}
		wg.Done()
	}()

	go func() {
		for {
			select {
			case v := <- queue.Get():
				fmt.Println("...Get", v)
			}
			time.Sleep(2 * time.Second)
		}
		wg.Done()
	}()

	wg.Wait()

}
