package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"time"
)

const (
	localhost     string = "localhost"
	localbaseport int    = 10000
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

func getHandler(obj GETReq) string {
	return string("get handler result")
}

func postHandler(obj POSTReq) string {
	return fmt.Sprintf("post handler:%d", obj.Id)
}

func putHandler(obj PUTReq) string {
	return fmt.Sprintf("put handler:%d", obj.Id)
}
func deleteHandler(obj DELETEReq) string {
	return fmt.Sprintf("delete handler:%d", obj.Id)
}

func commandHandler(obj APIObj) <-chan interface{} {

	log.Printf("command handler %#v", obj)

	ch := make(chan interface{})
	go func() {
		defer close(ch)

		switch obj := obj.(type) {
		case GETReq:
			ch <- getHandler(obj)
		case POSTReq:
			ch <- postHandler(obj)
		case PUTReq:
			ch <- putHandler(obj)
		case DELETEReq:
			ch <- deleteHandler(obj)
		default:
			ch <- string("unsuppot obj")
		}
	}()
	return ch
}

func main() {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, os.Interrupt, os.Kill)

	startBridge(ctx, string(localhost+":10000"), string(localhost+":20000"), string(localhost+":10002"), string(localhost+":20002"))
	startEchoSession(ctx, string(localhost+":10004"), string(localhost+":20004"))

	APIStart(ctx, commandHandler)

	var once sync.Once
	for {
		once.Do(func() { log.Print("start") })

		select {
		case s := <-sigc:
			log.Println("signal received", s)
			cancel()
			time.Sleep(1 * time.Second)
			return
		case <-time.After(10 * time.Second):
			log.Println("timeout 1")
		}
	}
}
