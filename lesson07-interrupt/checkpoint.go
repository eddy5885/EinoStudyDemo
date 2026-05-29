package main

import (
	"context"
	"sync"
)

// memoryCheckPointStore 保存 Runner 中断时的检查点（本课用内存，进程退出即失）。
type memoryCheckPointStore struct {
	mu sync.Mutex
	m  map[string][]byte
}

func newMemoryCheckPointStore() *memoryCheckPointStore {
	return &memoryCheckPointStore{m: make(map[string][]byte)}
}

func (s *memoryCheckPointStore) Set(_ context.Context, key string, value []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[key] = value
	return nil
}

func (s *memoryCheckPointStore) Get(_ context.Context, key string) ([]byte, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.m[key]
	return v, ok, nil
}
