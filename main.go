package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"time"
	"sync"
)

const (
	localhost string = "localhost"
	localbaseport int = 10000
)

func startEchoSession(ctx context.Context, local1, remote1 string) (*Dialogue, error) {

	session1, err := createSession(local1, remote1)
	if err != nil {
		return nil, errors.New("")
	}

	dialogue, err := createDialogue(session1)
	if err != nil {
		return nil, errors.New("")
	}

	dialogue.start(context.WithValue(ctx, "echo", true))

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

	session.streams = make(map[uint32]*RTPStream)
	stream := new(RTPStream)
	session.streams[0] = stream // まずはSSRC０のみ使う

	return session, nil
}

func createDialogue(sessions ...*RTPSession) (*Dialogue, error) {

	dialogue := new(Dialogue)

	for _, s := range sessions {
		dialogue.sessions = append(dialogue.sessions, s)
	}

	return dialogue, nil
}

func commandHandler(c chan<- interface{}, obj interface{}) {

	go func() {
		c <- string("hello world")
	}()
}

func main() {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, os.Interrupt, os.Kill)

	startBridge(ctx, string(localhost+":10000"), string(localhost+":20000"), string(localhost+":10002"), string(localhost+":20002"))
	startEchoSession(ctx, string(localhost+":10004"), string(localhost+":20004"))

	apireq := APIStart(ctx)

	var once sync.Once
	for {
		once.Do(func() { log.Print("start") })

		select {
		case obj, ok := <- apireq:
			log.Print("api req received")
			if !ok {
				log.Print("err received")
			} else {
				commandHandler(obj.(ReqObj).Ch,
					obj.(ReqObj).AnyReq)
			}
		case s := <- sigc:
			log.Println("signal received", s)
			cancel()
			time.Sleep(1 * time.Second)
			return
		case <- time.After(10 * time.Second):
			log.Println("timeout 1")
		}
	}
}
