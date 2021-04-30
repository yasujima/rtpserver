package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/nokute78/go-bit/pkg/bit/v2"
	"log"
	"net"
	"os"
	"os/signal"
	"time"
	"sync"
)

type event struct {
	buf    []byte
	val    string
	source net.Addr
	self   *RTPSession
}

type RTPStream struct {
	Ssrc uint32
	Ts   uint32
	Seq  uint16
}

type RTPSession struct {
	streams map[uint32]*RTPStream

	recvqueue *Queue

	remoteAddr *net.UDPAddr
	localAddr  *net.UDPAddr
}


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
	} ()

	return multiplexedStream
}

func (session *RTPSession) getRecvQ() *Queue {
	return session.recvqueue
}

func (session *RTPSession) createSendingData(eve *event) ([]byte, error) {

	header := RTPHeader{}
	br := bytes.NewReader(eve.buf)
	log.Println("reader len=", br.Len())
	if err := bit.Read(br, binary.BigEndian, &header); err != nil {
		return []byte{}, errors.New("binary read fail")
	}
	var ssrc uint32 = 0

	var seq uint16 = session.streams[0].Seq
	session.streams[0].Seq += 1
	var ts uint32 = session.streams[0].Ts
	session.streams[0].Ts += uint32(br.Len())

	h := createHeader(0, seq, ts, ssrc)
	var chunk [1500]byte
	s, err := br.Read(chunk[:])
	if err != nil {
		log.Println("br read fail!!!", err)
	}
	ret := append(h, chunk[:s]...)
	return ret, nil

}

type Dialogue struct {
	sessions []*RTPSession
}

var localhost string = "localhost"
var localbaseport int = 10000

func (session *RTPSession) start(ctx context.Context, dialoguepipe <-chan *event) chan *event {

	con, _ := net.ListenUDP("udp", session.localAddr)

	receiverWork := func(ctx context.Context) *Queue {
		qchan := make(chan interface{})
		queue := newQueue(ctx, qchan)
		go func() { //receiver work
			defer close(qchan)
			var buf [1500]byte
			for {
				con.SetDeadline(time.Now().Add(3 * time.Second))
				n, remote, err := con.ReadFromUDP(buf[:])
				if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
					continue
				}
				log.Println("num..", n)
				log.Println("remt.", remote)
				log.Println("val..", string(buf[:]))
				e := event{self: session}
				e.buf = make([]byte, n)
				copy(e.buf, buf[:])
				e.source = remote
				select {
				case queue.In() <- &e:
				case <-ctx.Done():
					log.Println("canceled receiver")
					return
				}
			}
		}()
		return queue
	}

	bridgeWork := func(ctx context.Context, queue *Queue) chan *event {
		pipe := make(chan *event)
		go func() { // get and bridge work
			defer close(pipe)
			for {
				select {
				case v := <-queue.Out():
					pipe <- v.(*event)
				case <-ctx.Done():
					log.Println("canceled putter")
					return
				}
			}
		}()
		return pipe
	}

	senderWork := func(ctx context.Context, pipe <-chan *event) {

		echo := ctx.Value("echo")
		
		go func() { // sender work
			for {
				select {
				case v := <-pipe:
					if echo == true || v.self != session {
						data, _ := session.createSendingData(v)
						con.WriteTo(data, session.remoteAddr)
						log.Printf("write to %v", data, " addr", session.remoteAddr)
					}
				case <-ctx.Done():
					log.Println("canceled writer")
					con.Close()
					return
				}
			}
		}()
	}

	queue := receiverWork(ctx)
	sessionpipe := bridgeWork(ctx, queue)
	senderWork(ctx, dialoguepipe)

	return sessionpipe
}

func startEchoSession(ctx context.Context, local1, remote1 string) (*Dialogue, error) {

	session1, err := createSession(local1, remote1)
	if err != nil {
		return nil, errors.New("")
	}

	ctx = context.WithValue(ctx, "echo", true)

	dialogue, err := createDialogue(session1)
	if err != nil {
		return nil, errors.New("")
	}

	dialogue.start(ctx)

	return dialogue, nil
}

func startBridge(ctx context.Context, local1, remote1, local2, remote2 string) (*Dialogue, error) {

	session1, err := createSession(local1, remote1)
	if err != nil {
		return nil, errors.New("")
	}

	session2, err := createSession(local2, remote2)
	if err != nil {
		return nil, errors.New("")
	}

	dialogue, err := createDialogue(session1, session2)
	if err != nil {
		return nil, errors.New("")
	}

	dialogue.start(ctx)

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

	//	qchan := make(chan<- interface{})
	//	session.recvqueue = newQueue(ctx, qchan)

	session.streams = make(map[uint32]*RTPStream)
	stream := new(RTPStream)
	session.streams[0] = stream // まずはSSRC０のみ使う

	//	session.pipe = make(chan *event)

	log.Printf(".... %#v\n", session)

	return session, nil
}

func createDialogue(sessions ...*RTPSession) (*Dialogue, error) {

	dialogue := new(Dialogue)

 	for _, s := range sessions {
		dialogue.sessions = append(dialogue.sessions, s)
	}


	return dialogue, nil
}

func (dialogue *Dialogue) start(ctx context.Context) {
	
	sendPipes := []chan *event{}
	recvPipes := []<-chan *event{}
 	for _, s := range dialogue.sessions {
		spipe := make(chan *event)
		rpipe := s.start(ctx, spipe)
		sendPipes = append(sendPipes, spipe)
		recvPipes = append(recvPipes, rpipe)
	}

	

	go func() { //dialogue goroutine
		defer func () {
			for _, c := range sendPipes {
				close(c)
			}
		}()
		
		for {
			select {
			case v := <-fanIn(ctx, recvPipes...):
				for _, sp := range sendPipes {
					sp<- v
				}
			case <-ctx.Done():
				log.Println("dialogue routine cannceled")
				return
			}
		}
	}()
}


func main() {

	//	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, os.Interrupt, os.Kill)

	startBridge(ctx, string(localhost+":10000"), string(localhost+":20000"), string(localhost+":10002"), string(localhost+":20002"))

	startEchoSession(ctx, string(localhost+":10004"), string(localhost+":20004"))

	for {
		select {
		case s := <-sigc:
			log.Println("signal received", s)
			cancel()
			time.Sleep(1*time.Second)
			return
		case <-time.After(1 * time.Second):
			log.Println("timeout 1")
		}
	}
}
