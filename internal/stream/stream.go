// Package stream is an in-process pub/sub registry for per-user state pushes.
// Actors publish a user's latest state here; the StreamState RPC handler
// subscribes a connected client to them. It replaces the old websocket
// connection registry.
package stream

import (
	"sync"

	"cityio/internal/domain"
	"cityio/internal/metrics"
)

// StateUpdate is a per-user snapshot pushed to subscribers.
// Conversion to proto happens at the RPC boundary. Only non-nil fields are sent.
type StateUpdate struct {
	User              *domain.User
	City              *domain.City
	Building          *domain.Building
	DeletedBuildingID *string
}

type subscriber struct {
	id uint64
	ch chan StateUpdate
}

var (
	mu     sync.Mutex
	subs   = make(map[string][]subscriber)
	nextID uint64
)

// Subscribe registers a subscriber for a user's state pushes and returns the
// receive channel plus an unsubscribe function. The channel is buffered and
// drops the oldest pending value on overflow so a slow client never blocks a
// publisher.
func Subscribe(userID string) (<-chan StateUpdate, func()) {
	mu.Lock()
	defer mu.Unlock()

	nextID++
	s := subscriber{id: nextID, ch: make(chan StateUpdate, 8)}
	subs[userID] = append(subs[userID], s)
	metrics.StreamSubscribers.Inc()

	unsubscribe := func() {
		mu.Lock()
		defer mu.Unlock()
		list := subs[userID]
		for i, existing := range list {
			if existing.id == s.id {
				subs[userID] = append(list[:i], list[i+1:]...)
				metrics.StreamSubscribers.Dec()
				break
			}
		}
		if len(subs[userID]) == 0 {
			delete(subs, userID)
		}
		close(s.ch)
	}

	return s.ch, unsubscribe
}

// Publish delivers a state update to every subscriber of the user. It never
// blocks: if a subscriber's buffer is full the oldest value is discarded to
// make room for the newest.
func Publish(userID string, state StateUpdate) {
	recordPublish(state)

	mu.Lock()
	defer mu.Unlock()

	for _, s := range subs[userID] {
		select {
		case s.ch <- state:
		default:
			// Buffer full — drop oldest, then try again. If we still can't
			// enqueue, count the drop and move on.
			metrics.StreamBufferDropsTotal.Inc()
			select {
			case <-s.ch:
			default:
			}
			select {
			case s.ch <- state:
			default:
				metrics.StreamBufferDropsTotal.Inc()
			}
		}
	}
}

// recordPublish bumps the stream publish counter once per populated field on
// the update. A single update can carry user + city + building, so each
// non-nil field is counted independently.
func recordPublish(state StateUpdate) {
	if state.User != nil {
		metrics.StreamPublishesTotal.WithLabelValues("user").Inc()
	}
	if state.City != nil {
		metrics.StreamPublishesTotal.WithLabelValues("city").Inc()
	}
	if state.Building != nil {
		metrics.StreamPublishesTotal.WithLabelValues("building").Inc()
	}
	if state.DeletedBuildingID != nil {
		metrics.StreamPublishesTotal.WithLabelValues("deletion").Inc()
	}
}
