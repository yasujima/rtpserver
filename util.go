package main

import (
	"context"
	"sync"
)

func fanIn(
	ctx context.Context,
	channels ...<-chan *event,
) <-chan *event {
	var wg sync.WaitGroup
	multiplexedStream := make(chan *event)

	multiplex := func(c <-chan *event) {
		defer wg.Done()
		for i := range c {
			select {
			case <- ctx.Done():
				return
			case multiplexedStream <- i:
			}
		}
	}

	wg.Add(len(channels))
	for _, c := range channels {
		go multiplex(c)
	}

	go func() {
		wg.Wait()
		close(multiplexedStream)
	}()

	return multiplexedStream
}
