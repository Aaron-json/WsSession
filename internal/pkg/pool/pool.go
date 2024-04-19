package pool

import (
	"sync"
)

type Pool[K comparable, V any] struct {
	mu     sync.RWMutex
	values map[K]V
	max    int
}

// Creates a new pool. If max < 0, the returned pool is unbounded
func NewPool[K comparable, V any](max int) *Pool[K, V] {
	return &Pool[K, V]{
		mu:     sync.RWMutex{},
		values: make(map[K]V),
		max:    max,
	}
}
func (p *Pool[K, V]) Delete(key K) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.values, key)
}

func (p *Pool[K, V]) DeleteAll(key K) {
	p.mu.Lock()
	defer p.mu.Unlock()
	clear(p.values)
}

// Store() guarantees that on error, no change will be made to the
// pool
func (p *Pool[K, V]) Store(key K, value V) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, exists := p.values[key]; exists {
		return DUPLICATE_KEY
	}
	if len(p.values) >= p.max && p.max >= 0 {
		return MAX_CAPACITY
	}
	p.values[key] = value
	return nil
}

// update guarantees that if an error occurs, the change to pool
// never happens
func (p *Pool[K, V]) Update(key K, value V) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	_, ok := p.values[key]
	if !ok {
		return KEY_NOT_FOUND
	}
	p.values[key] = value
	return nil
}

func (p *Pool[K, V]) Get(key K) (V, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	val, ok := p.values[key]
	if !ok {
		return *new(V), KEY_NOT_FOUND
	}
	return val, nil
}

func (p *Pool[K, V]) Exists(key K) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	_, ok := p.values[key]
	return ok
}

func (p *Pool[K, V]) Size() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.values)
}
