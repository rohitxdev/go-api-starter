package util

import "sync"

func NewSet[T comparable](initial ...T) *set[T] {
	m := make(map[T]struct{}, len(initial))
	for _, v := range initial {
		m[v] = struct{}{}
	}

	return &set[T]{
		items: m,
	}
}

type set[T comparable] struct {
	items map[T]struct{}
	mu    sync.RWMutex
}

func (s *set[T]) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.items)
}

func (s *set[T]) Has(item T) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, ok := s.items[item]
	return ok
}

func (s *set[T]) Items() []T {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]T, 0, len(s.items))
	for k := range s.items {
		out = append(out, k)
	}

	return out
}

func (s *set[T]) Add(items ...T) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.items == nil {
		s.items = make(map[T]struct{})
	}

	for _, v := range items {
		s.items[v] = struct{}{}
	}
}

func (s *set[T]) Remove(items ...T) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, v := range items {
		delete(s.items, v)
	}
}

func (s *set[T]) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.items = make(map[T]struct{})
}
