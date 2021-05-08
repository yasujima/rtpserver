package main

import (
	"sync"
	"log"
)

type ConcurrentKeyValueStore struct {
	m  map[string]interface{}
	mu sync.RWMutex
}

func NewConcurrentKeyValueStore() *ConcurrentKeyValueStore {
	return &ConcurrentKeyValueStore{m: make(map[string]interface{})}
}

func (s *ConcurrentKeyValueStore) Set(k, v string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[k] = v
}

func (s *ConcurrentKeyValueStore) Get(k string) (interface{}, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.m[k]
	return v, ok
}

func main() {

	store := NewConcurrentKeyValueStore()

	store.Set("aaa", "111")
	store.Set("bbb", "222")

	v, b := store.Get("aaa")
	log.Printf(".. %v: %v", v, b)

	v, b = store.Get("bbb")
	log.Printf(".. %v: %v", v, b)
	
	v, b = store.Get("ccc")
	log.Printf(".. %v: %v", v, b)

}

	
