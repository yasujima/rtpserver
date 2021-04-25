package main

import (
	"fmt"
	"net"
	"context"
	//	"strings"
	"time"
	"errors"
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
}

type Dialogue struct {
	eventChannel chan *event
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

}


func (c *RTPChannel) start(ctx context.Context, queue chan<- *event) {

	con, _ := net.ListenUDP("udp", c.localAddr)

	go func() {

		var buf [1500]byte

		for {
			e := event{self:c}			
			con.SetDeadline(time.Now().Add(3 * time.Second))
			n, remote, err := con.ReadFromUDP(buf[:])
			if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
                fmt.Printf("connection timeout : %v", con.RemoteAddr())
				e.val = "timeout"
			}
			fmt.Println("..", n)		
			fmt.Println("..", remote)
			fmt.Println("..", err)
			e.buf = make([]byte, n)
			copy(e.buf, buf[:])
			e.addr = remote
			select {
			case queue <- &e:
			case <-ctx.Done():
				fmt.Println("canceled")					
				return
			}
		}
	}()

	go func() {
		for {
			select {
			// case v := <-get():
			// 	con.WriteTo(v.buf, v.addr)
			// 	fmt.Println("write to ", string(v.buf) , " addr" , v.addr)			
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

	chan1.start(ctx, dialogue.eventChannel)
	chan2.start(ctx, dialogue.eventChannel)
	
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

	fmt.Printf(".... %#v\n", channel)

	
	return channel, nil
}

func createDialogue(ctx context.Context, channels... *RTPChannel) (*Dialogue, error) {

	dialogue := new(Dialogue)
	for _, c := range channels {
		dialogue.channels = append(dialogue.channels, c)
	}
	eventChan := make(chan *event)
	dialogue.eventChannel = eventChan

	go func(ctx context.Context) {
		for {
			select {
			case v := <-eventChan:
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

	startBridge(ctx, string("127.0.0.1:10000"), string("127.0.0.1:20000"))
	
	for {
		select {
		case <-time.After(1 * time.Second):
			fmt.Println("timeout 1")
		}
	}
}
	
