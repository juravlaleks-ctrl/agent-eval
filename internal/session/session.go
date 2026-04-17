package session

import (
	"sync"
	"time"
)

// State holds the in-memory state for one active eval session.
// Persisted state lives in SQLite; this tracks what we need between calls.
type State struct {
	SessionID        string
	CurrentTaskID    string   // "" when no task is currently active
	TasksRemaining   []string // task IDs still to be served, in order
	LastActivity     time.Time
	AwaitingSubmit   bool // true after eval_next, false after eval_submit
	ConsecutiveNexts int  // increments on each eval_next while AwaitingSubmit is true
}

// Manager holds all active in-memory session states and provides
// goroutine-safe access.
type Manager struct {
	mu       sync.Mutex
	sessions map[string]*State
	timeout  time.Duration
}

// NewManager creates a Manager with the given idle timeout.
func NewManager(idleTimeout time.Duration) *Manager {
	return &Manager{
		sessions: make(map[string]*State),
		timeout:  idleTimeout,
	}
}

// Get returns the State for sessionID and a bool indicating whether it exists.
// The returned pointer must not be accessed without calling Update; use it
// only for a snapshot read while holding no lock.
func (m *Manager) Get(sessionID string) (*State, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.sessions[sessionID]
	if !ok {
		return nil, false
	}
	// Return a copy to avoid unsynchronised access.
	copy := *s
	return &copy, true
}

// Create registers a new session with the given task IDs and returns a pointer
// to the newly created State. Overwrites any existing state for the same ID.
func (m *Manager) Create(sessionID string, taskIDs []string) *State {
	m.mu.Lock()
	defer m.mu.Unlock()
	ids := make([]string, len(taskIDs))
	copy(ids, taskIDs)
	s := &State{
		SessionID:      sessionID,
		TasksRemaining: ids,
		LastActivity:   time.Now(),
	}
	m.sessions[sessionID] = s
	return s
}

// Update calls fn with the session state under the manager lock, allowing
// callers to mutate state safely. Does nothing if sessionID does not exist.
func (m *Manager) Update(sessionID string, fn func(*State)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if s, ok := m.sessions[sessionID]; ok {
		fn(s)
	}
}

// Delete removes a session from the in-memory map.
func (m *Manager) Delete(sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, sessionID)
}

// ReapIdle removes and returns the IDs of sessions whose LastActivity exceeds
// the idle timeout. The caller is responsible for marking them abandoned in the
// persistent store.
func (m *Manager) ReapIdle() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	var reaped []string
	for id, s := range m.sessions {
		if time.Since(s.LastActivity) > m.timeout {
			reaped = append(reaped, id)
			delete(m.sessions, id)
		}
	}
	return reaped
}
