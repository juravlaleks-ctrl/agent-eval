package tasks

import (
	"crypto/rand"
	"embed"
	"fmt"
	"io/fs"
	"math/big"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

//go:embed tasks.yaml
var tasksFS embed.FS

var (
	hashHexPattern = regexp.MustCompile(`^[0-9a-fA-F]{64}$`)
	idPattern      = regexp.MustCompile(`^T\d{3}$`)
)

// Task is one evaluation prompt plus its grading specification.
type Task struct {
	ID         string    `yaml:"id"`
	Category   string    `yaml:"category"`
	Difficulty int       `yaml:"difficulty"`
	Prompt     string    `yaml:"prompt"`
	Source     string    `yaml:"source"`
	License    string    `yaml:"license"`
	Check      CheckSpec `yaml:"check"`
}

// CheckSpec describes how a task answer is validated.
type CheckSpec struct {
	Type      string     `yaml:"type"`
	Normalize string     `yaml:"normalize,omitempty"`
	Expected  string     `yaml:"expected,omitempty"`
	Positive  []string   `yaml:"positive,omitempty"`
	Negative  []string   `yaml:"negative,omitempty"`
	Groups    [][]string `yaml:"groups,omitempty"`
}

// Corpus is the validated in-memory task set.
type Corpus struct {
	tasks []Task
	byID  map[string]Task
}

type corpusFile struct {
	Tasks []Task `yaml:"tasks"`
}

// Load reads the embedded tasks.yaml file and validates the corpus.
func Load() (*Corpus, error) {
	data, err := fs.ReadFile(tasksFS, "tasks.yaml")
	if err != nil {
		return nil, fmt.Errorf("tasks: read embedded tasks.yaml: %w", err)
	}

	var doc corpusFile
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("tasks: parse tasks.yaml: %w", err)
	}

	if err := validate(doc.Tasks); err != nil {
		return nil, err
	}

	tasks := append([]Task(nil), doc.Tasks...)
	sortTasks(tasks)

	byID := make(map[string]Task, len(tasks))
	for _, task := range tasks {
		byID[task.ID] = task
	}

	return &Corpus{
		tasks: tasks,
		byID:  byID,
	}, nil
}

// Pick returns task IDs matching difficulty range [minDiff, maxDiff] inclusive.
// Order is category, then difficulty, then ID. If randomize is true, a
// crypto/rand Fisher-Yates shuffle is applied to the filtered IDs.
func (c *Corpus) Pick(minDiff, maxDiff int, randomize bool) []string {
	if minDiff < 1 || maxDiff > 3 || minDiff > maxDiff {
		return nil
	}

	ids := make([]string, 0, len(c.tasks))
	for _, task := range c.tasks {
		if task.Difficulty < minDiff || task.Difficulty > maxDiff {
			continue
		}
		ids = append(ids, task.ID)
	}

	if randomize {
		shuffleIDs(ids)
	}
	return ids
}

// Get returns a task by ID.
func (c *Corpus) Get(id string) (Task, bool) {
	task, ok := c.byID[id]
	return task, ok
}

// All returns all tasks in stable corpus order.
func (c *Corpus) All() []Task {
	return append([]Task(nil), c.tasks...)
}

func validate(tasks []Task) error {
	if len(tasks) != 54 {
		return fmt.Errorf("tasks: expected 54 tasks, got %d", len(tasks))
	}

	allowedCategories := map[string]bool{
		"arithmetic":  true,
		"code":        true,
		"instruction": true,
		"logic":       true,
		"multi_step":  true,
		"tool_use":    true,
	}

	seenIDs := make(map[string]bool, len(tasks))
	categoryCounts := make(map[string]int)
	cellCounts := make(map[string]int)

	for i, task := range tasks {
		if !idPattern.MatchString(task.ID) {
			return fmt.Errorf("tasks: task[%d]: invalid id %q", i, task.ID)
		}
		if seenIDs[task.ID] {
			return fmt.Errorf("tasks: duplicate id %q", task.ID)
		}
		seenIDs[task.ID] = true

		if !allowedCategories[task.Category] {
			return fmt.Errorf("tasks: %s: unknown category %q", task.ID, task.Category)
		}
		if task.Difficulty < 1 || task.Difficulty > 3 {
			return fmt.Errorf("tasks: %s: difficulty %d out of range [1, 3]", task.ID, task.Difficulty)
		}
		if strings.TrimSpace(task.Prompt) == "" {
			return fmt.Errorf("tasks: %s: prompt must not be empty", task.ID)
		}
		if strings.TrimSpace(task.Source) == "" {
			return fmt.Errorf("tasks: %s: source must not be empty", task.ID)
		}
		if strings.TrimSpace(task.License) == "" {
			return fmt.Errorf("tasks: %s: license must not be empty", task.ID)
		}
		if err := validateCheck(task); err != nil {
			return err
		}

		categoryCounts[task.Category]++
		cellKey := fmt.Sprintf("%s/%d", task.Category, task.Difficulty)
		cellCounts[cellKey]++
	}

	for i := 1; i <= 54; i++ {
		id := fmt.Sprintf("T%03d", i)
		if !seenIDs[id] {
			return fmt.Errorf("tasks: missing id %q", id)
		}
	}

	for category := range allowedCategories {
		if categoryCounts[category] != 9 {
			return fmt.Errorf("tasks: category %q: expected 9 tasks, got %d", category, categoryCounts[category])
		}
		for difficulty := 1; difficulty <= 3; difficulty++ {
			cellKey := fmt.Sprintf("%s/%d", category, difficulty)
			if cellCounts[cellKey] != 3 {
				return fmt.Errorf(
					"tasks: category %q difficulty %d: expected 3 tasks, got %d",
					category,
					difficulty,
					cellCounts[cellKey],
				)
			}
		}
	}

	return nil
}

func validateCheck(task Task) error {
	switch task.Check.Type {
	case "hash":
		if task.Check.Normalize != "lower_trim" {
			return fmt.Errorf("tasks: %s: hash check requires normalize=lower_trim", task.ID)
		}
		if !hashHexPattern.MatchString(task.Check.Expected) {
			return fmt.Errorf("tasks: %s: hash check requires 64 hex chars in expected", task.ID)
		}
	case "regex_functional":
		if len(task.Check.Positive) == 0 {
			return fmt.Errorf("tasks: %s: regex_functional requires non-empty positive cases", task.ID)
		}
	case "lambda_python":
		if len(task.Check.Positive) == 0 || len(task.Check.Negative) == 0 {
			return fmt.Errorf("tasks: %s: lambda_python requires non-empty positive and negative cases", task.ID)
		}
		for _, value := range append(append([]string(nil), task.Check.Positive...), task.Check.Negative...) {
			if _, err := strconv.Atoi(strings.TrimSpace(value)); err != nil {
				return fmt.Errorf("tasks: %s: lambda_python test value %q is not an int", task.ID, value)
			}
		}
	case "keywords":
		if len(task.Check.Groups) == 0 {
			return fmt.Errorf("tasks: %s: keywords requires at least one group", task.ID)
		}
		for i, group := range task.Check.Groups {
			if len(group) == 0 {
				return fmt.Errorf("tasks: %s: keywords group %d must not be empty", task.ID, i)
			}
		}
	default:
		return fmt.Errorf("tasks: %s: unsupported check type %q", task.ID, task.Check.Type)
	}
	return nil
}

func sortTasks(tasks []Task) {
	sort.Slice(tasks, func(i, j int) bool {
		if tasks[i].Category != tasks[j].Category {
			return tasks[i].Category < tasks[j].Category
		}
		if tasks[i].Difficulty != tasks[j].Difficulty {
			return tasks[i].Difficulty < tasks[j].Difficulty
		}
		return tasks[i].ID < tasks[j].ID
	})
}

func shuffleIDs(ids []string) {
	for i := len(ids) - 1; i > 0; i-- {
		jBig, err := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		if err != nil {
			return
		}
		j := int(jBig.Int64())
		ids[i], ids[j] = ids[j], ids[i]
	}
}
