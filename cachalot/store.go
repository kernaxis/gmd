package cachalot

import "sync"

// store is a generic, mutex-protected in-memory map keyed by Docker object ID.
// It backs the containers and images caches, and can be reused as-is for any
// future resource type (volumes, networks, ...).
type store[T any] struct {
	mu   sync.RWMutex
	data map[string]T
}

func newStore[T any]() *store[T] {
	return &store[T]{data: make(map[string]T)}
}

func (s *store[T]) set(id string, v T) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[id] = v
}

func (s *store[T]) delete(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, id)
}

func (s *store[T]) get(id string) (T, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.data[id]
	return v, ok
}

func (s *store[T]) list() []T {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]T, 0, len(s.data))
	for _, v := range s.data {
		out = append(out, v)
	}
	return out
}
