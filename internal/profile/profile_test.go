package profile

import (
	"strings"
	"testing"

	"agent/internal/config"
)

func makeProfiles() []config.Profile {
	return []config.Profile{
		{Match: config.ProfileMatch{Model: "exact-model", Effort: "low"}, Difficulty: [2]int{1, 1}},
		{Match: config.ProfileMatch{Model: "claude-opus-*", Effort: "low"}, Difficulty: [2]int{1, 3}},
		{Match: config.ProfileMatch{Model: "claude-opus-*", Effort: "high"}, Difficulty: [2]int{3, 3}},
		{Match: config.ProfileMatch{Model: "claude-sonnet-*", Effort: "medium"}, Difficulty: [2]int{1, 3}},
	}
}

func TestExactMatchWinsOverWildcard(t *testing.T) {
	// Add a wildcard that would also match "exact-model" to verify exact takes priority.
	profiles := append(makeProfiles(),
		config.Profile{Match: config.ProfileMatch{Model: "exact-*", Effort: "low"}, Difficulty: [2]int{2, 3}},
	)
	m := NewMatcher(profiles)

	diff, err := m.Match("exact-model", "low")
	if err != nil {
		t.Fatalf("Match: %v", err)
	}
	if diff != [2]int{1, 1} {
		t.Errorf("exact match: got %v, want [1 1]", diff)
	}
}

func TestWildcardSuffixMatch(t *testing.T) {
	m := NewMatcher(makeProfiles())

	diff, err := m.Match("claude-opus-4-7", "low")
	if err != nil {
		t.Fatalf("Match: %v", err)
	}
	if diff != [2]int{1, 3} {
		t.Errorf("wildcard match: got %v, want [1 3]", diff)
	}

	diff, err = m.Match("claude-opus-4-7", "high")
	if err != nil {
		t.Fatalf("Match: %v", err)
	}
	if diff != [2]int{3, 3} {
		t.Errorf("wildcard high match: got %v, want [3 3]", diff)
	}
}

func TestWildcardDoesNotMatchUnrelatedPrefix(t *testing.T) {
	m := NewMatcher(makeProfiles())

	_, err := m.Match("gpt-4-turbo", "low")
	if err == nil {
		t.Fatal("expected error for unmatched model")
	}
}

func TestNoMatchReturnsExactErrorFormat(t *testing.T) {
	m := NewMatcher(makeProfiles())

	_, err := m.Match("unknown-model", "medium")
	if err == nil {
		t.Fatal("expected error")
	}

	want := `unknown (model, effort) pair: model="unknown-model", effort="medium"; add profile to config.yaml`
	if err.Error() != want {
		t.Errorf("error message mismatch:\ngot:  %s\nwant: %s", err.Error(), want)
	}
}

func TestEffortMismatch(t *testing.T) {
	m := NewMatcher(makeProfiles())

	_, err := m.Match("claude-opus-4-7", "medium") // no medium profile for opus
	if err == nil {
		t.Fatal("expected error for missing effort variant")
	}
	if !strings.Contains(err.Error(), "add profile to config.yaml") {
		t.Errorf("expected hint in error, got: %s", err.Error())
	}
}
