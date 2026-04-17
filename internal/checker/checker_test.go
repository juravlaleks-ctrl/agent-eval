package checker

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os/exec"
	"strings"
	"testing"
)

func TestHashPass(t *testing.T) {
	c := New()
	answer := "  Hello World  "
	normalised := strings.TrimSpace(strings.ToLower(answer))
	sum := sha256.Sum256([]byte(normalised))
	expected := hex.EncodeToString(sum[:])

	r := c.Check(context.Background(), answer, Check{Type: "hash", Expected: expected})
	if !r.Passed {
		t.Errorf("expected pass, got reason: %s", r.Reason)
	}
}

func TestHashFail(t *testing.T) {
	c := New()
	r := c.Check(context.Background(), "hello", Check{Type: "hash", Expected: "deadbeef"})
	if r.Passed {
		t.Error("expected fail")
	}
	if r.Reason == "" {
		t.Error("expected non-empty reason on failure")
	}
}

func TestHashCaseInsensitive(t *testing.T) {
	c := New()
	answer := "test"
	sum := sha256.Sum256([]byte(answer))
	expectedUpper := strings.ToUpper(hex.EncodeToString(sum[:]))

	r := c.Check(context.Background(), answer, Check{Type: "hash", Expected: expectedUpper})
	if !r.Passed {
		t.Errorf("expected pass for uppercase expected hex, got reason: %s", r.Reason)
	}
}

func TestRegexPass(t *testing.T) {
	c := New()
	r := c.Check(context.Background(), `\d+`, Check{
		Type:     "regex_functional",
		Positive: []string{"123", "0", "9999"},
		Negative: []string{"abc", "12x", ""},
	})
	if !r.Passed {
		t.Errorf("expected pass, got reason: %s", r.Reason)
	}
}

func TestRegexFailPositive(t *testing.T) {
	c := New()
	r := c.Check(context.Background(), `\d+`, Check{
		Type:     "regex_functional",
		Positive: []string{"abc"},
	})
	if r.Passed {
		t.Error("expected fail: regex should not match 'abc'")
	}
	if !strings.Contains(r.Reason, "did not fullmatch") {
		t.Errorf("unexpected reason: %s", r.Reason)
	}
}

func TestRegexFailNegative(t *testing.T) {
	c := New()
	r := c.Check(context.Background(), `.*`, Check{
		Type:     "regex_functional",
		Negative: []string{"anything"},
	})
	if r.Passed {
		t.Error("expected fail: .* should match everything")
	}
	if !strings.Contains(r.Reason, "unexpectedly matched") {
		t.Errorf("unexpected reason: %s", r.Reason)
	}
}

func TestRegexInvalidPattern(t *testing.T) {
	c := New()
	r := c.Check(context.Background(), `[invalid`, Check{
		Type:     "regex_functional",
		Positive: []string{"x"},
	})
	if r.Passed {
		t.Error("expected fail for invalid regex")
	}
	if !strings.Contains(r.Reason, "invalid regex") {
		t.Errorf("unexpected reason: %s", r.Reason)
	}
}

func TestKeywordsPass(t *testing.T) {
	c := New()
	r := c.Check(context.Background(), "The quick brown fox jumps over the lazy dog", Check{
		Type: "keywords",
		Groups: [][]string{
			{"fox", "cat"},
			{"dog", "wolf"},
			{"QUICK"},
		},
	})
	if !r.Passed {
		t.Errorf("expected pass, got reason: %s", r.Reason)
	}
}

func TestKeywordsFailMissingGroup(t *testing.T) {
	c := New()
	r := c.Check(context.Background(), "hello world", Check{
		Type: "keywords",
		Groups: [][]string{
			{"hello"},
			{"python", "golang"}, // neither present
		},
	})
	if r.Passed {
		t.Error("expected fail")
	}
	if !strings.Contains(r.Reason, "missing keyword group 1") {
		t.Errorf("unexpected reason: %s", r.Reason)
	}
}

func TestKeywordsEmptyGroups(t *testing.T) {
	c := New()
	r := c.Check(context.Background(), "anything", Check{
		Type:   "keywords",
		Groups: [][]string{},
	})
	if !r.Passed {
		t.Errorf("no groups should always pass, got reason: %s", r.Reason)
	}
}

func TestLambdaPythonSkipIfUnavailable(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not in PATH")
	}
}

func TestLambdaPythonPass(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not in PATH")
	}
	c := New()
	r := c.Check(context.Background(), "lambda x: x * 2", Check{
		Type:     "lambda_python",
		Positive: []string{"2", "5"},  // fn(2)=4 truthy, fn(5)=10 truthy
		Negative: []string{"0"},       // fn(0)=0 falsy
	})
	if !r.Passed {
		t.Errorf("expected pass, got reason: %s", r.Reason)
	}
}

func TestLambdaPythonFailPositive(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not in PATH")
	}
	c := New()
	// lambda always returns 0 (falsy), positive case should fail
	r := c.Check(context.Background(), "lambda x: 0", Check{
		Type:     "lambda_python",
		Positive: []string{"1"},
	})
	if r.Passed {
		t.Error("expected fail")
	}
}

func TestLambdaPythonFailNegative(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not in PATH")
	}
	c := New()
	// lambda always returns 1 (truthy), negative case should fail
	r := c.Check(context.Background(), "lambda x: 1", Check{
		Type:     "lambda_python",
		Negative: []string{"5"},
	})
	if r.Passed {
		t.Error("expected fail")
	}
}

func TestLambdaPythonNoPython(t *testing.T) {
	c := &Checker{pythonPath: ""}
	r := c.Check(context.Background(), "lambda x: x", Check{
		Type:     "lambda_python",
		Positive: []string{"1"},
	})
	if r.Passed {
		t.Error("expected fail when python not available")
	}
	if r.Reason != "python3 not available" {
		t.Errorf("unexpected reason: %s", r.Reason)
	}
}

func TestUnknownCheckType(t *testing.T) {
	c := New()
	r := c.Check(context.Background(), "x", Check{Type: "unknown"})
	if r.Passed {
		t.Error("expected fail for unknown type")
	}
}
