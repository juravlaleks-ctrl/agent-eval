package tasks

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"agent/internal/checker"
)

func TestLoad(t *testing.T) {
	corpus := mustLoad(t)
	if got := len(corpus.All()); got != 54 {
		t.Fatalf("len(All()): got %d, want 54", got)
	}
	if _, ok := corpus.Get("T054"); !ok {
		t.Fatal("Get(T054): expected task to exist")
	}
}

func TestCorpusShape(t *testing.T) {
	corpus := mustLoad(t)

	categoryCounts := make(map[string]int)
	cellCounts := make(map[string]int)
	for _, task := range corpus.All() {
		categoryCounts[task.Category]++
		cellCounts[fmt.Sprintf("%s/%d", task.Category, task.Difficulty)]++
	}

	for _, category := range []string{"arithmetic", "code", "instruction", "logic", "multi_step", "tool_use"} {
		if got := categoryCounts[category]; got != 9 {
			t.Fatalf("category %s count: got %d, want 9", category, got)
		}
		for difficulty := 1; difficulty <= 3; difficulty++ {
			key := fmt.Sprintf("%s/%d", category, difficulty)
			if got := cellCounts[key]; got != 3 {
				t.Fatalf("%s count: got %d, want 3", key, got)
			}
		}
	}
}

func TestPickOrdering(t *testing.T) {
	corpus := mustLoad(t)

	wantAll := make([]string, 0, 54)
	for i := 1; i <= 54; i++ {
		wantAll = append(wantAll, fmt.Sprintf("T%03d", i))
	}

	if got := corpus.Pick(1, 3, false); !reflect.DeepEqual(got, wantAll) {
		t.Fatalf("Pick(1,3,false):\ngot  %v\nwant %v", got, wantAll)
	}

	wantMid := make([]string, 0, 36)
	for _, task := range corpus.All() {
		if task.Difficulty >= 2 {
			wantMid = append(wantMid, task.ID)
		}
	}
	if got := corpus.Pick(2, 3, false); !reflect.DeepEqual(got, wantMid) {
		t.Fatalf("Pick(2,3,false):\ngot  %v\nwant %v", got, wantMid)
	}

	wantEasy := make([]string, 0, 18)
	for _, task := range corpus.All() {
		if task.Difficulty == 1 {
			wantEasy = append(wantEasy, task.ID)
		}
	}
	if got := corpus.Pick(1, 1, false); !reflect.DeepEqual(got, wantEasy) {
		t.Fatalf("Pick(1,1,false):\ngot  %v\nwant %v", got, wantEasy)
	}
}

func TestCheckerIntegration(t *testing.T) {
	corpus := mustLoad(t)
	if got := len(knownCorrectAnswers); got != 54 {
		t.Fatalf("knownCorrectAnswers: got %d entries, want 54", got)
	}

	checks := checker.New()
	ctx := context.Background()

	for _, task := range corpus.All() {
		answer, ok := knownCorrectAnswers[task.ID]
		if !ok {
			t.Fatalf("missing known correct answer for %s", task.ID)
		}
		if task.Check.Type == "lambda_python" && !checks.PythonAvailable() {
			t.Skip("python3 not available")
		}

		result := checks.Check(ctx, answer, checker.Check{
			Type:     task.Check.Type,
			Expected: task.Check.Expected,
			Positive: append([]string(nil), task.Check.Positive...),
			Negative: append([]string(nil), task.Check.Negative...),
			Groups:   append([][]string(nil), task.Check.Groups...),
		})
		if !result.Passed {
			t.Fatalf("%s (%s) failed checker integration: %s", task.ID, task.Check.Type, result.Reason)
		}
	}
}

func mustLoad(t *testing.T) *Corpus {
	t.Helper()

	corpus, err := Load()
	if err != nil {
		t.Fatalf("Load(): %v", err)
	}
	return corpus
}
