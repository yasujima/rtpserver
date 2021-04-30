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

	go func() { //dialogue goroutine
		defer func() {
			for _, spipe := range sendPipes {
				close(spipe)
			}
		}()

		for {
			select {
			case v := <-fanIn(ctx, recvPipes...):
				for _, sp := range sendPipes {
					sp <- v
				}
			case <-ctx.Done():
				log.Println("dialogue routine cannceled")
				return
			}
		}
	}()
}
