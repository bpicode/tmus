package core

import "sync"

// topic provides a thread-safe publish-subscribe mechanism for events of type T.
type topic[T any] struct {
	mu   sync.Mutex
	subs map[chan T]struct{}
}

// newTopic creates and initializes a new topic.
func newTopic[T any]() *topic[T] {
	return &topic[T]{
		subs: make(map[chan T]struct{}),
	}
}

// subscribe returns a channel to receive events and a function to unsubscribe.
// The channel has a small buffer (8) to accommodate bursts of events.
func (t *topic[T]) subscribe() (<-chan T, func()) {
	ch := make(chan T, 8)
	t.mu.Lock()
	t.subs[ch] = struct{}{}
	t.mu.Unlock()
	return ch, func() { t.unsubscribe(ch) }
}

// broadcast sends the event to all active subscribers.
// If a subscriber's channel buffer is full, the event is silently dropped
// for that subscriber to prevent blocking the publisher.
func (t *topic[T]) broadcast(event T) {
	t.mu.Lock()
	defer t.mu.Unlock()
	for ch := range t.subs {
		select {
		case ch <- event:
		default:
		}
	}
}

// unsubscribe removes the given channel from subscriptions and closes it.
func (t *topic[T]) unsubscribe(ch chan T) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if _, ok := t.subs[ch]; ok {
		delete(t.subs, ch)
		close(ch)
	}
}

// close closes all active subscriptions and clears them.
func (t *topic[T]) close() {
	t.mu.Lock()
	defer t.mu.Unlock()
	for ch := range t.subs {
		close(ch)
	}
	t.subs = make(map[chan T]struct{})
}
