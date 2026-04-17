package checker

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// Check is the specification for a single evaluation check.
type Check struct {
	Type     string     // "hash" | "regex_functional" | "lambda_python" | "keywords"
	Expected string     // used by hash
	Positive []string   // regex_functional: must fullmatch; lambda_python: truthy; keywords: unused
	Negative []string   // regex_functional: must NOT match; lambda_python: falsy; keywords: unused
	Groups   [][]string // keywords: each group must contribute at least one match
}

// Result is the outcome of a single check.
type Result struct {
	Passed bool
	Reason string // non-empty on failure
}

// Checker dispatches check executions.
type Checker struct {
	pythonPath string // path to python3, or "" if not found
}

// New creates a Checker and detects python3 in PATH.
func New() *Checker {
	path, err := exec.LookPath("python3")
	if err != nil {
		path = ""
	}
	return &Checker{pythonPath: path}
}

// PythonAvailable reports whether python3 was found at construction time.
func (c *Checker) PythonAvailable() bool {
	return c.pythonPath != ""
}

// Check evaluates answer against spec and returns a Result.
func (c *Checker) Check(ctx context.Context, answer string, spec Check) Result {
	switch spec.Type {
	case "hash":
		return c.checkHash(answer, spec)
	case "regex_functional":
		return c.checkRegex(answer, spec)
	case "lambda_python":
		return c.checkLambdaPython(ctx, answer, spec)
	case "keywords":
		return c.checkKeywords(answer, spec)
	default:
		return Result{Passed: false, Reason: fmt.Sprintf("unknown check type %q", spec.Type)}
	}
}

// checkHash computes SHA-256 of the lowercased, trimmed answer and compares it
// to spec.Expected (case-insensitive hex comparison).
func (c *Checker) checkHash(answer string, spec Check) Result {
	normalised := strings.TrimSpace(strings.ToLower(answer))
	sum := sha256.Sum256([]byte(normalised))
	got := hex.EncodeToString(sum[:])
	if strings.EqualFold(got, spec.Expected) {
		return Result{Passed: true}
	}
	return Result{Passed: false, Reason: fmt.Sprintf("hash mismatch: got %s", got)}
}

// checkRegex compiles answer as a Go regular expression and verifies it
// fullmatches all Positive strings and none of the Negative strings.
// FullMatch is implemented by anchoring the pattern: `^(?:pattern)$`.
func (c *Checker) checkRegex(answer string, spec Check) Result {
	anchored := `^(?:` + answer + `)$`
	re, err := regexp.Compile(anchored)
	if err != nil {
		return Result{Passed: false, Reason: fmt.Sprintf("invalid regex: %v", err)}
	}
	for _, s := range spec.Positive {
		if !re.MatchString(s) {
			return Result{Passed: false, Reason: fmt.Sprintf("regex did not fullmatch %q", s)}
		}
	}
	for _, s := range spec.Negative {
		if re.MatchString(s) {
			return Result{Passed: false, Reason: fmt.Sprintf("regex unexpectedly matched %q", s)}
		}
	}
	return Result{Passed: true}
}

// checkLambdaPython evaluates a Python lambda expression against integer
// test values via python3.
func (c *Checker) checkLambdaPython(ctx context.Context, answer string, spec Check) Result {
	if c.pythonPath == "" {
		return Result{Passed: false, Reason: "python3 not available"}
	}

	parseInts := func(strs []string) ([]int, error) {
		ints := make([]int, 0, len(strs))
		for _, s := range strs {
			n, err := strconv.Atoi(strings.TrimSpace(s))
			if err != nil {
				return nil, fmt.Errorf("non-numeric test value: %q", s)
			}
			ints = append(ints, n)
		}
		return ints, nil
	}

	pos, err := parseInts(spec.Positive)
	if err != nil {
		return Result{Passed: false, Reason: err.Error()}
	}
	neg, err := parseInts(spec.Negative)
	if err != nil {
		return Result{Passed: false, Reason: err.Error()}
	}

	posJSON, _ := json.Marshal(pos)
	negJSON, _ := json.Marshal(neg)

	// The script evaluates the lambda with builtins disabled for basic safety,
	// then exits 1 if any positive case fails or 2 if any negative case passes.
	const script = `import sys,json; fn=eval(sys.argv[1], {'__builtins__': {}}, {}); pos=json.loads(sys.argv[2]); neg=json.loads(sys.argv[3]); [sys.exit(1) for x in pos if not fn(x)]; [sys.exit(2) for x in neg if fn(x)]; sys.exit(0)`

	ctxTimeout, cancel := context.WithTimeout(ctx, 3*1e9)
	defer cancel()

	cmd := exec.CommandContext(ctxTimeout, c.pythonPath, "-c", script,
		answer, string(posJSON), string(negJSON))
	cmd.Env = []string{"PATH=" + os.Getenv("PATH")}

	out, err := cmd.CombinedOutput()
	if err == nil {
		return Result{Passed: true}
	}

	if ctxTimeout.Err() != nil {
		return Result{Passed: false, Reason: "lambda_python: execution timed out"}
	}

	if exitErr, ok := err.(*exec.ExitError); ok {
		switch exitErr.ExitCode() {
		case 1:
			return Result{Passed: false, Reason: "lambda returned falsy for a positive test case"}
		case 2:
			return Result{Passed: false, Reason: "lambda returned truthy for a negative test case"}
		}
	}

	return Result{Passed: false, Reason: fmt.Sprintf("lambda_python error: %v; output: %s", err, strings.TrimSpace(string(out)))}
}

// checkKeywords verifies that the lowercased answer contains at least one
// keyword from each group.
func (c *Checker) checkKeywords(answer string, spec Check) Result {
	lower := strings.ToLower(answer)
	for i, group := range spec.Groups {
		matched := false
		for _, kw := range group {
			if strings.Contains(lower, strings.ToLower(kw)) {
				matched = true
				break
			}
		}
		if !matched {
			return Result{Passed: false, Reason: fmt.Sprintf("missing keyword group %d", i)}
		}
	}
	return Result{Passed: true}
}
