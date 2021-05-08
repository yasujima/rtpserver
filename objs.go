package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"github.com/nokute78/go-bit/pkg/bit/v2"
	"log"
	"net"
	"time"
)

type AddressObj struct {
	Address string
	Port int
}

type StreamObj struct {
	Realm string
	RemoteAddr AddressObj
	Pt int
}
	
type SessionObj struct {
	Streams map[int]*StreamObj
}

type ReqObj struct {
	Ch chan <- interface{}
	AnyReq interface{}
}

type POSTReq struct {
	Id int
	Target string
	Sessions map[int]*SessionObj
}

type POSTResp struct {
	Id int
	Result bool
	Sessions map[int]*SessionObj
}

type PUTReq struct {
	Id int
	Sessions map[int]*SessionObj
}

type PUTResp struct {
	Id int
	Sessions map[int]*SessionObj
}

type GETReq struct {
	Id int
}

type GETResp struct {
	Id int
}

type DELETEReq struct {
	Id int
}

type DELETEResp struct {
	id int
}

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
	streams    map[uint32]*RTPStream
	recvQueue  *Queue
	remoteAddr *net.UDPAddr
	localAddr  *net.UDPAddr
}

type Dialogue struct {
	sessions []*RTPSession
}

//////////////////////////////////////////////////
// RTPSession
//////////////////////////////////////////////////
func (session *RTPSession) getRecvQ() *Queue {
	return session.recvQueue
}

func (session *RTPSession) createSendingData(eve *event) ([]byte, error) {

	header := RTPHeader{}
	br := bytes.NewReader(eve.buf)
	if err := bit.Read(br, binary.BigEndian, &header); err != nil {
		return nil, errors.New("header read fail")
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
		return nil, errors.New("chunk read fail")
	}
	ret := append(h, chunk[:s]...)
	return ret, nil
}

func (session *RTPSession) start(ctx context.Context, dialoguepipe <-chan *event) chan *event {

	con, _ := net.ListenUDP("udp", session.localAddr)

	receiverWork := func(ctx context.Context) *Queue {
		rpipe := make(chan interface{})
		queue := newQueue(ctx, rpipe)
		go func() { //receiver work
			defer close(rpipe)
			var buf [1500]byte
			for {
				con.SetDeadline(time.Now().Add(3 * time.Second))
				n, remote, err := con.ReadFromUDP(buf[:])
				if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
					continue
				}
				e := event{self: session}
				e.buf = make([]byte, n)
				copy(e.buf, buf[:])
				e.source = remote
				select {
				case queue.Put() <- &e:
				case <- ctx.Done():
					log.Print("canceled receiver")
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
				case v := <- queue.Get():
					if v != nil {
						pipe <- v.(*event)
					}
				case <- ctx.Done():
					log.Print("canceled putter")
					return
				}
			}
		}()
		return pipe
	}

	senderWork := func(ctx context.Context, pipe <-chan *event) {

		echo := ctx.Value("echo")

		go func() { // sender work
			defer con.Close()
			for {
				select {
				case v := <- pipe:
					if (v != nil) && (echo == true || v.self != session) {
						data, _ := session.createSendingData(v)
						con.WriteTo(data, session.remoteAddr)
					}
				case <- ctx.Done():
					log.Print("canceled writer")
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

//////////////////////////////////////////////////
// Dialogue
//////////////////////////////////////////////////
func (dialogue *Dialogue) start(ctx context.Context) {

	sendPipes := []chan *event{}
	recvPipes := []<-chan *event{}
	for _, s := range dialogue.sessions {
		spipe := make(chan *event)
		rpipe := s.start(ctx, spipe)
		sendPipes = append(sendPipes, spipe)
		recvPipes = append(recvPipes, rpipe)
	}

	mixed := fanIn(ctx, recvPipes...)

	go func() { //dialogue goroutine
		defer func() {
			for _, spipe := range sendPipes {
				close(spipe)
			}
		}()

		for {
			select {
			case v := <- mixed:
				for _, sp := range sendPipes {
					sp <- v
				}
			case <- ctx.Done():
				log.Println("dialogue routine cannceled")
				return
			}
		}
	}()
}
