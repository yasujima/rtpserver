package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"time"
)

type RTPStream struct {
	queue *Queue
	// ssrc etc
}

type RTPSession struct {
	streams []*RTPStream
}

func (session *RTPSession) getStreamQ() *Queue {
	return session.streams[0].queue
}

type RTPChannel struct {
	remoteAddr *net.UDPAddr
	localAddr  *net.UDPAddr
	session    *RTPSession
	pipe       chan *event
}

type Dialogue struct {
	pipe     chan *event
	channels []*RTPChannel
}

type event struct {
	buf  []byte
	val  string
	addr net.Addr
	self *RTPChannel
}

var localhost string = "localhost"
var localbaseport int = 10000

func (c *RTPChannel) put(eve *event) {
	c.pipe <- eve
}

func (c *RTPChannel) start(ctx context.Context, pipe chan<- *event) {

	con, _ := net.ListenUDP("udp", c.localAddr)

	go func() {

		var buf [1500]byte
		for {

			con.SetDeadline(time.Now().Add(3 * time.Second))
			n, remote, err := con.ReadFromUDP(buf[:])
			if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
				continue
			}
			fmt.Println("num..", n)
			fmt.Println("rad..", remote)
			fmt.Println("err..", err)
			fmt.Println("val..", string(buf[:]))
			e := event{self: c}
			e.buf = make([]byte, n)
			copy(e.buf, buf[:])
			e.addr = remote
			select {
			case c.session.getStreamQ().In() <- &e:
			case <-ctx.Done():
				fmt.Println("canceled receiver")
				return
			}
		}
	}()

	go func() {
		for {
			select {
			case v := <-c.session.getStreamQ().Out():
				pipe <- v.(*event)
			case <-ctx.Done():
				fmt.Println("canceled putter")
				return
			}
		}
	}()

	go func() {
		for {
			select {
			case v := <-c.pipe:
				con.WriteTo(v.buf, c.remoteAddr)
				fmt.Println("write to ", string(v.buf), " addr", c.remoteAddr)
			case <-ctx.Done():
				fmt.Println("canceled writer")
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
	stream.queue = newQueue()
	session := new(RTPSession)
	session.streams = append(session.streams, stream)
	channel.session = session

	channel.pipe = make(chan *event)

	fmt.Printf(".... %#v\n", channel)

	return channel, nil
}

func createDialogue(ctx context.Context, channels ...*RTPChannel) (*Dialogue, error) {

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
				fmt.Println("dialogue routine cannceled")
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

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, os.Interrupt, os.Kill)

	startBridge(ctx, string(localhost+":10000"), string(localhost+":20000"))

	for {
		select {
		case s := <-sigc:
			fmt.Println("signal received", s)
			return
		case <-time.After(1 * time.Second):
			fmt.Println("timeout 1")
		}
	}
}
