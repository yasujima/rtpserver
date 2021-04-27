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

	remoteAddr *net.UDPAddr
	localAddr  *net.UDPAddr
	pipe       chan *event
}

func (session *RTPSession) getStreamQ() *Queue {
	return session.streams[0].queue
}

type Dialogue struct {
	pipe     chan *event
	sessions []*RTPSession
}

type event struct {
	buf    []byte
	val    string
	source net.Addr
	self   *RTPSession
}

var localhost string = "localhost"
var localbaseport int = 10000

func (c *RTPSession) put(eve *event) {
	c.pipe <- eve
}

func (c *RTPSession) start(ctx context.Context, pipe chan<- *event) {

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
			fmt.Println("remt.", remote)
			fmt.Println("err..", err)
			fmt.Println("val..", string(buf[:]))
			e := event{self: c}
			e.buf = make([]byte, n)
			copy(e.buf, buf[:])
			e.source = remote
			select {
			case c.getStreamQ().In() <- &e:
			case <-ctx.Done():
				fmt.Println("canceled receiver")
				return
			}
		}
	}()

	go func() {
		for {
			select {
			case v := <-c.getStreamQ().Out():
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

	session1, err := createSession(local, remote)
	if err != nil {
		return nil, errors.New("")
	}

	session2, err := createSession(remote, local)
	if err != nil {
		return nil, errors.New("")
	}

	dialogue, err := createDialogue(ctx, session1, session2)
	if err != nil {
		return nil, errors.New("")
	}

	session1.start(ctx, dialogue.pipe)
	session2.start(ctx, dialogue.pipe)

	return dialogue, nil
}

func createSession(local, remote string) (*RTPSession, error) {

	laddr, err := net.ResolveUDPAddr("udp", local)
	if err != nil {
		return nil, fmt.Errorf("local addr invalid", local)
	}
	raddr, err := net.ResolveUDPAddr("udp", remote)
	if err != nil {
		return nil, fmt.Errorf("remote addr invalid", remote)
	}
	session := new(RTPSession)
	session.localAddr = laddr
	session.remoteAddr = raddr

	stream := new(RTPStream)
	stream.queue = newQueue()
	session.streams = append(session.streams, stream)

	session.pipe = make(chan *event)

	fmt.Printf(".... %#v\n", session)

	return session, nil
}

func createDialogue(ctx context.Context, sessions ...*RTPSession) (*Dialogue, error) {

	dialogue := new(Dialogue)
	for _, s := range sessions {
		dialogue.sessions = append(dialogue.sessions, s)
	}
	pipe := make(chan *event)
	dialogue.pipe = pipe

	go func(ctx context.Context) { //dialogue goroutine
		for {
			select {
			case v := <-pipe:
				for _, s := range dialogue.sessions {
					if v.self != s {
						s.put(v)
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
