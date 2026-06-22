// Package checkstream is an in-process SSE event hub. Each check has one
// Channel; the orchestrator emits events into it and subscribers receive
// them over HTTP. Late subscribers can request replay from a given seq.
//
// Channels are pinned for 10 minutes after the check completes so a slow
// browser tab can still catch up. Memory is bounded by a 256-event ring.
package checkstream

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	ringSize       = 256
	completedHold  = 10 * time.Minute
)

// Event is a single SSE payload.
type Event struct {
	Seq       int            `json:"seq"`
	Stage     string         `json:"stage"`            // 'init','resolve','extract','screen','retrieval','investigator','judge','done','error'
	Progress  int            `json:"progress"`         // 0..100
	Message   string         `json:"message,omitempty"`
	ClaimID   string         `json:"claim_id,omitempty"`
	Payload   map[string]any `json:"payload,omitempty"`
	Timestamp time.Time      `json:"ts"`
}

// Channel is the per-check stream.
type Channel struct {
	checkID     uuid.UUID
	mu          sync.RWMutex
	buf         []Event // ring buffer, oldest first; bounded by ringSize
	nextSeq     int     // monotonically increasing — survives ring wrap
	subscribers map[chan Event]struct{}
	closed      bool
	closedAt    time.Time
}

// Hub owns all live channels.
type Hub struct {
	mu       sync.Mutex
	channels map[uuid.UUID]*Channel
}

func NewHub() *Hub {
	h := &Hub{channels: map[uuid.UUID]*Channel{}}
	go h.janitor()
	return h
}

// Get returns the channel for checkID, creating it if absent.
// Returns nil for the zero UUID (defensive — callers should check).
func (h *Hub) Get(checkID uuid.UUID) *Channel {
	if checkID == uuid.Nil {
		return nil
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	ch, ok := h.channels[checkID]
	if !ok {
		ch = &Channel{
			checkID:     checkID,
			buf:         make([]Event, 0, ringSize),
			subscribers: map[chan Event]struct{}{},
		}
		h.channels[checkID] = ch
	}
	return ch
}

// janitor drops channels that completed > completedHold ago.
func (h *Hub) janitor() {
	t := time.NewTicker(1 * time.Minute)
	defer t.Stop()
	for now := range t.C {
		h.mu.Lock()
		for id, ch := range h.channels {
			ch.mu.RLock()
			drop := ch.closed && now.Sub(ch.closedAt) > completedHold
			ch.mu.RUnlock()
			if drop {
				ch.shutdownSubs()
				delete(h.channels, id)
			}
		}
		h.mu.Unlock()
	}
}

// --- Channel ops ---------------------------------------------------------

// Emit appends an event to the ring and fans out to subscribers.
// Non-blocking — if a subscriber is slow, the event is dropped for that sub.
// Seq is monotonically increasing across the full channel lifetime, NOT a
// ring index. After the buffer wraps, new events keep climbing; replay-from-N
// subscribers see only events with Seq > N (older events that fell out of
// the ring are gone, but never confused with newer events at the same index).
func (c *Channel) Emit(e Event) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return
	}
	e.Seq = c.nextSeq
	c.nextSeq++
	if len(c.buf) >= ringSize {
		copy(c.buf, c.buf[1:])
		c.buf = c.buf[:len(c.buf)-1]
	}
	c.buf = append(c.buf, e)
	for sub := range c.subscribers {
		select {
		case sub <- e:
		default:
			// drop on slow subscriber — replay covers it if they reconnect.
		}
	}
}

// Close marks the channel as completed. After this, no more Emits are
// accepted but subscribers can replay until janitor cleanup.
func (c *Channel) Close() {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return
	}
	c.closed = true
	c.closedAt = time.Now()
	for sub := range c.subscribers {
		close(sub)
	}
	c.subscribers = map[chan Event]struct{}{}
}

func (c *Channel) shutdownSubs() {
	c.mu.Lock()
	defer c.mu.Unlock()
	for sub := range c.subscribers {
		close(sub)
	}
	c.subscribers = map[chan Event]struct{}{}
}

// Subscribe returns a channel that receives all future events plus replay
// of any events with Seq > lastSeen. The returned cancel function unsubscribes.
func (c *Channel) Subscribe(lastSeen int) (<-chan Event, func()) {
	if c == nil {
		ch := make(chan Event)
		close(ch)
		return ch, func() {}
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	sub := make(chan Event, 16)
	if !c.closed {
		c.subscribers[sub] = struct{}{}
	}
	// Replay anything missed.
	for _, e := range c.buf {
		if e.Seq > lastSeen {
			select {
			case sub <- e:
			default:
			}
		}
	}
	if c.closed {
		close(sub)
	}
	cancel := func() {
		c.mu.Lock()
		defer c.mu.Unlock()
		if _, ok := c.subscribers[sub]; ok {
			delete(c.subscribers, sub)
			close(sub)
		}
	}
	return sub, cancel
}
