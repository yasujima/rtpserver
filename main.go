package main

import (
	"fmt"
	"net"
	"context"
	//	"strings"
	"time"
	"errors"
	"os"
	"os/signal"
)

type RTPStream struct {
	queue []*event
	// ssrc etc
}

type RTPSession struct {

	streams []*RTPStream
}


type RTPChannel struct {
	remoteAddr *net.UDPAddr
	localAddr *net.UDPAddr
	// e.t.c.
	session *RTPSession
	pipe chan *event
}

type Dialogue struct {
	pipe chan *event
	channels []*RTPChannel
}

type event struct {
	buf []byte
	val string
	addr net.Addr
	self *RTPChannel
}

var localhost string = "localhost"
var localbaseport int = 10000

func (c *RTPChannel) put(eve *event) {
	c.pipe<-eve
}


func (c *RTPChannel) start(ctx context.Context, pipe chan<- *event) {

	con, _ := net.ListenUDP("udp", c.localAddr)

	go func() {//change this to 2 goroutine sandwitch with event queue.

		var buf [1500]byte

		for {
			e := event{self:c}			
			con.SetDeadline(time.Now().Add(3 * time.Second))
			n, remote, err := con.ReadFromUDP(buf[:])
			if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
				e.val = "timeout"
				continue
			}
			fmt.Println("num..", n)		
			fmt.Println("rad..", remote)
			fmt.Println("err..", err)
			fmt.Println("val..", string(buf[:]))
			e.buf = make([]byte, n)
			copy(e.buf, buf[:])
			e.addr = remote
			select {
			case pipe <- &e:
			case <-ctx.Done():
				fmt.Println("canceled")					
				return
			}
		}
	}()

	go func() {
		for {
			select {
			case v := <-c.pipe:
			 	con.WriteTo(v.buf, c.remoteAddr)
			 	fmt.Println("write to ", string(v.buf) , " addr" , c.remoteAddr)			
			case <-ctx.Done():
				fmt.Println("canceled")
				return
			}
		}
	}()

	
}

func startBridge(ctx context.Context, local, remote string) (*Dialogue, error) {

	chan1, err := createChannel(local, remote)
	if err != nil {
		return nil, errors.New("")
	}

	chan2, err := createChannel(remote, local)
	if err != nil {
		return nil, errors.New("")
	}

	dialogue, err := createDialogue(ctx, chan1, chan2)
	if err != nil {
		return nil, errors.New("")
	}

	chan1.start(ctx, dialogue.pipe)
	chan2.start(ctx, dialogue.pipe)
	
	return dialogue, nil
}

func createChannel(local, remote string) (*RTPChannel, error) {

	laddr, err := net.ResolveUDPAddr("udp", local)
	if err != nil {
		return nil, fmt.Errorf("local addr invalid", local)
	}
	raddr, err := net.ResolveUDPAddr("udp", remote)
	if err != nil {
		return nil, fmt.Errorf("remote addr invalid", remote)
	}
	channel := new(RTPChannel)
	channel.localAddr = laddr
	channel.remoteAddr = raddr

	stream := new(RTPStream)
	stream.queue = make([]*event, 0)
	session := new(RTPSession)
	session.streams = append(session.streams, stream)
	channel.session = session

	channel.pipe = make(chan *event)

	fmt.Printf(".... %#v\n", channel)

	
	return channel, nil
}

func createDialogue(ctx context.Context, channels... *RTPChannel) (*Dialogue, error) {

	dialogue := new(Dialogue)
	for _, c := range channels {
		dialogue.channels = append(dialogue.channels, c)
	}
	pipe := make(chan *event)
	dialogue.pipe = pipe

	go func(ctx context.Context) { //dialogue goroutine
		for {
			select {
			case v := <-pipe:
				for _, c := range dialogue.channels {
					if v.self != c {
						c.put(v)
					}
				}
			case <-ctx.Done():
				return
			}
		}
	}(ctx)
	
	return dialogue, nil
}


func main() {

	//	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)

	startBridge(ctx, string("127.0.0.1:10000"), string("127.0.0.1:20000"))
	
	for {
		select {
		case s := <-c:
			fmt.Println("signal received", s)
			return
		case <-time.After(1 * time.Second):
			fmt.Println("timeout 1")
		}
	}
}
	
