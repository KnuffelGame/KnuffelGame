package models

import (
	"fmt"
	"sync"
	"time"
)

// TargetType identifies the type of stream target.
type TargetType string

const (
	// lobby target type for lobby streams
	TargetTypeLobby TargetType = "lobby"
	// TargetTypeGame removed in MVP - only lobby streams supported
)

// TargetKey is the composite key used in the registry map.
type TargetKey string

// MakeTargetKey composes a registry key like "lobby:{id}".
func MakeTargetKey(tt TargetType, id string) TargetKey {
	switch tt {
	case TargetTypeLobby:
		return TargetKey(fmt.Sprintf("lobby:%s", id))
	default:
		// Fallback to raw string to avoid surprises
		return TargetKey(fmt.Sprintf("%s:%s", string(tt), id))
	}
}

// SSEMessage represents an internal dispatch message to be sent to connections.
type SSEMessage struct {
	Event string
	Data  []byte
}

// Connection represents a single client connection for SSE streaming.
type Connection struct {
	UserID string
	Ch     chan SSEMessage
	Done   chan struct{}
}

// TargetEntry holds metadata and connections for a specific target (lobby or game).
type TargetEntry struct {
	Type        TargetType
	ID          string
	Connections map[string]*Connection
	CreatedAt   time.Time
	LastEventAt time.Time
	Metrics     struct {
		EventsSent        int
		FailedConnections int
	}
}

// Registry is a concurrent-safe registry of SSE targets and their connections.
type Registry struct {
	mu      sync.RWMutex
	targets map[TargetKey]*TargetEntry
}

// NewRegistry constructs a new empty Registry.
func NewRegistry() *Registry {
	return &Registry{
		targets: make(map[TargetKey]*TargetEntry),
	}
}

// EnsureTarget returns an existing target or creates a new one if missing.
// It is safe for concurrent use.
func (r *Registry) EnsureTarget(tt TargetType, id string) *TargetEntry {
	key := MakeTargetKey(tt, id)
	r.mu.Lock()
	defer r.mu.Unlock()

	if te, ok := r.targets[key]; ok {
		return te
	}

	te := &TargetEntry{
		Type:        tt,
		ID:          id,
		Connections: make(map[string]*Connection),
		CreatedAt:   time.Now().UTC(),
		LastEventAt: time.Time{},
	}
	r.targets[key] = te
	return te
}

// RegisterConnection creates channels and registers a connection for the given user on the target.
// Returns the target entry and the new connection.
func (r *Registry) RegisterConnection(tt TargetType, id, userID string) (*TargetEntry, *Connection) {
	te := r.EnsureTarget(tt, id)

	// Build the connection object
	conn := &Connection{
		UserID: userID,
		Ch:     make(chan SSEMessage, 16), // small buffer to absorb bursts
		Done:   make(chan struct{}, 1),    // signal-only
	}

	// Insert into connections map under lock
	r.mu.Lock()
	te.Connections[userID] = conn
	r.mu.Unlock()

	return te, conn
}

// RemoveConnection safely removes a user's connection from the target and signals completion.
// It is safe to call multiple times; subsequent calls will be no-ops.
func (r *Registry) RemoveConnection(tt TargetType, id, userID string) {
	key := MakeTargetKey(tt, id)

	// Remove under lock
	r.mu.Lock()
	te, ok := r.targets[key]
	if !ok {
		r.mu.Unlock()
		return
	}
	conn, exists := te.Connections[userID]
	if exists {
		delete(te.Connections, userID)
	}
	r.mu.Unlock()

	// Signal connection done (do not close conn.Ch to avoid send panics elsewhere)
	if exists && conn != nil {
		select {
		case conn.Done <- struct{}{}:
		default:
			// already signaled or receiver gone
		}
	}
}

// UnregisterTarget closes all connections for the target and removes it from the registry.
// Returns the number of connections closed and whether the target existed.
func (r *Registry) UnregisterTarget(tt TargetType, id string) (closed int, ok bool) {
	key := MakeTargetKey(tt, id)

	// Snapshot recipients under lock
	r.mu.Lock()
	te, exists := r.targets[key]
	if !exists {
		r.mu.Unlock()
		return 0, false
	}
	// Collect for cleanup
	var conns []*Connection
	for _, c := range te.Connections {
		conns = append(conns, c)
	}
	// Remove target entry
	delete(r.targets, key)
	r.mu.Unlock()

	// Signal all connections done
	for _, c := range conns {
		select {
		case c.Done <- struct{}{}:
			closed++
		default:
			// already signaled or receiver gone
			closed++
		}
	}

	return closed, true
}

// Broadcast sends a message to all or a specific user's connection for a target.
// It collects recipients under lock then dispatches without holding locks.
// Returns: found (recipients discovered), sent (successful immediate sends), failed (dropped), ok (target existed).
func (r *Registry) Broadcast(tt TargetType, id string, msg SSEMessage, targetUserID string) (found int, sent int, failed int, ok bool) {
	key := MakeTargetKey(tt, id)

	// Collect recipients under read lock
	r.mu.RLock()
	te, exists := r.targets[key]
	if !exists {
		r.mu.RUnlock()
		return 0, 0, 0, false
	}

	var recipients []*Connection
	if targetUserID != "" {
		if c, has := te.Connections[targetUserID]; has {
			recipients = append(recipients, c)
		}
	} else {
		for _, c := range te.Connections {
			recipients = append(recipients, c)
		}
	}
	r.mu.RUnlock()

	found = len(recipients)

	// Dispatch outside locks using non-blocking sends
	for _, c := range recipients {
		select {
		case c.Ch <- msg:
			sent++
		default:
			// channel is full — consider this connection stalled; count failure and schedule cleanup
			failed++
			go r.RemoveConnection(tt, id, c.UserID)
		}
	}

	// Update metrics and last event timestamp under lock
	now := time.Now().UTC()
	r.mu.Lock()
	if te2, still := r.targets[key]; still {
		te2.LastEventAt = now
		te2.Metrics.EventsSent += sent
		te2.Metrics.FailedConnections += failed
	}
	r.mu.Unlock()

	return found, sent, failed, true
}

// Stats returns aggregate totals across the registry.
// Note: MVP only supports lobby targets, so gameCount and gameConnections will always be 0.
func (r *Registry) Stats() (totalTargets int, totalConnections int, lobbyCount int, lobbyConnections int, gameCount int, gameConnections int) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, te := range r.targets {
		totalTargets++
		connCount := len(te.Connections)
		totalConnections += connCount

		switch te.Type {
		case TargetTypeLobby:
			lobbyCount++
			lobbyConnections += connCount
		default:
			// MVP only supports lobby targets
		}
	}

	return
}

// HasTarget reports whether the given target exists in the registry.
// Note: helper for handlers; not part of original spec but useful for 409 semantics.
func (r *Registry) HasTarget(tt TargetType, id string) bool {
	key := MakeTargetKey(tt, id)
	r.mu.RLock()
	_, ok := r.targets[key]
	r.mu.RUnlock()
	return ok
}
