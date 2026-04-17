package profile

import (
	"fmt"
	"strings"

	"agent/internal/config"
)

// Matcher resolves (model, effort) pairs to difficulty ranges using a list of
// profiles. Exact model matches take precedence over wildcard matches.
type Matcher struct {
	profiles []config.Profile
}

// NewMatcher creates a Matcher backed by the given profile list.
func NewMatcher(profiles []config.Profile) *Matcher {
	return &Matcher{profiles: profiles}
}

// Match returns the difficulty range [min, max] for the given model and effort.
// Exact model match (case-sensitive) is tried first; wildcard suffix match
// second. Returns an error if no profile matches.
func (m *Matcher) Match(model, effort string) ([2]int, error) {
	// First pass: exact model match.
	for _, p := range m.profiles {
		if p.Match.Effort != effort {
			continue
		}
		if !strings.HasSuffix(p.Match.Model, "*") && p.Match.Model == model {
			return p.Difficulty, nil
		}
	}

	// Second pass: wildcard suffix match.
	for _, p := range m.profiles {
		if p.Match.Effort != effort {
			continue
		}
		if strings.HasSuffix(p.Match.Model, "*") {
			prefix := p.Match.Model[:len(p.Match.Model)-1]
			if strings.HasPrefix(model, prefix) {
				return p.Difficulty, nil
			}
		}
	}

	return [2]int{}, fmt.Errorf(
		"unknown (model, effort) pair: model=%q, effort=%q; add profile to config.yaml",
		model, effort,
	)
}
