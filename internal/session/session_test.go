package session

import (
	"testing"
	"time"
)

func TestCreateAndGet(t *testing.T) {
	m := NewManager(30 * time.Minute)

	s := m.Create("sess-1", []string{"t1", "t2", "t3"})
	if s.SessionID != "sess-1" {
		t.Errorf("SessionID: got %q, want sess-1", s.SessionID)
	}
	if len(s.TasksRemaining) != 3 {
		t.Errorf("TasksRemaining: got %d, want 3", len(s.TasksRemaining))
	}

	got, ok := m.Get("sess-1")
	if !ok {
		t.Fatal("Get: session not found")
	}
	if got.SessionID != "sess-1" {
		t.Errorf("Get SessionID: got %q", got.SessionID)
	}
}

func TestGetMissing(t *testing.T) {
	m := NewManager(30 * time.Minute)
	_, ok := m.Get("no-such")
	if ok {
		t.Error("expected ok=false for missing session")
	}
}

func TestUpdate(t *testing.T) {
	m := NewManager(30 * time.Minute)
	m.Create("sess-upd", []string{"t1"})

	m.Update("sess-upd", func(s *State) {
		s.AwaitingSubmit = true
		s.ConsecutiveNexts = 2
		s.CurrentTaskID = "t1"
	})

	got, ok := m.Get("sess-upd")
	if !ok {
		t.Fatal("session not found after update")
	}
	if !got.AwaitingSubmit {
		t.Error("expected AwaitingSubmit=true")
	}
	if got.ConsecutiveNexts != 2 {
		t.Errorf("ConsecutiveNexts: got %d, want 2", got.ConsecutiveNexts)
	}
	if got.CurrentTaskID != "t1" {
		t.Errorf("CurrentTaskID: got %q, want t1", got.CurrentTaskID)
	}
}

func TestUpdateMissing(t *testing.T) {
	m := NewManager(30 * time.Minute)
	// Should not panic on missing session.
	m.Update("no-such", func(s *State) {
		t.Error("fn should not be called for missing session")
	})
}

func TestDelete(t *testing.T) {
	m := NewManager(30 * time.Minute)
	m.Create("sess-del", []string{})
	m.Delete("sess-del")

	_, ok := m.Get("sess-del")
	if ok {
		t.Error("expected session to be deleted")
	}
}

func TestReapIdle(t *testing.T) {
	m := NewManager(10 * time.Millisecond)

	m.Create("active", []string{})
	m.Create("idle-1", []string{})
	m.Create("idle-2", []string{})

	// Backdate idle sessions.
	m.mu.Lock()
	m.sessions["idle-1"].LastActivity = time.Now().Add(-1 * time.Second)
	m.sessions["idle-2"].LastActivity = time.Now().Add(-1 * time.Second)
	m.sessions["active"].LastActivity = time.Now().Add(1 * time.Hour) // far future
	m.mu.Unlock()

	reaped := m.ReapIdle()
	if len(reaped) != 2 {
		t.Errorf("expected 2 reaped, got %d: %v", len(reaped), reaped)
	}

	// Verify active session is still present.
	_, ok := m.Get("active")
	if !ok {
		t.Error("active session should not have been reaped")
	}
	// Verify idle sessions are gone.
	for _, id := range []string{"idle-1", "idle-2"} {
		if _, ok := m.Get(id); ok {
			t.Errorf("session %q should have been reaped", id)
		}
	}
}

func TestReapIdleNone(t *testing.T) {
	m := NewManager(1 * time.Hour)
	m.Create("fresh", []string{})
	reaped := m.ReapIdle()
	if len(reaped) != 0 {
		t.Errorf("expected no reaps, got %d", len(reaped))
	}
}

func TestGetReturnsCopy(t *testing.T) {
	m := NewManager(30 * time.Minute)
	m.Create("sess-copy", []string{"t1"})

	got, _ := m.Get("sess-copy")
	got.AwaitingSubmit = true // mutate the copy

	got2, _ := m.Get("sess-copy")
	if got2.AwaitingSubmit {
		t.Error("Get should return a copy; mutation should not affect stored state")
	}
}
