package ai

import (
	"fmt"
	"io"
	"os"
	"time"
)

// Observer receives notifications during story execution.
type Observer interface {
	OnStoryStart(index, total int, story UserStory)
	OnStep(index, total int, result StepResult)
	OnStoryDone(index, total int, story UserStory)
}

// noopObserver is used when no observer is configured.
type noopObserver struct{}

func (noopObserver) OnStoryStart(_, _ int, _ UserStory) {}
func (noopObserver) OnStep(_, _ int, _ StepResult)      {}
func (noopObserver) OnStoryDone(_, _ int, _ UserStory)  {}

// ConsoleObserver prints live progress to the given writer.
//
// Example output:
//
//	[1/4] Create user happy path
//	      → sql_exec      {"query":"TRUNCATE users"}          → {"rows_affected":0}
//	      → http_request  {"method":"POST","path":"/users"}   → {"status":201,...}
//	      ✓ passed  (2 steps, 1.2s)
//
//	[2/4] Get user not found
//	      → http_request  {"method":"GET","path":"/users/99"} → {"status":404}
//	      ✓ passed  (1 steps, 320ms)
type ConsoleObserver struct {
	w io.Writer
}

// NewConsoleObserver creates an observer that prints progress to w.
// Defaults to os.Stdout if no writer is provided.
func NewConsoleObserver(w ...io.Writer) *ConsoleObserver {
	out := io.Writer(os.Stdout)
	if len(w) > 0 {
		out = w[0]
	}
	return &ConsoleObserver{w: out}
}

func (o *ConsoleObserver) OnStoryStart(index, total int, story UserStory) {
	fmt.Fprintf(o.w, "\n[%d/%d] %s\n", index+1, total, story.Title)
}

func (o *ConsoleObserver) OnStep(_ int, _ int, step StepResult) {
	in := truncate(step.Input, 42)
	out := truncate(step.Output, 32)
	fmt.Fprintf(o.w, "      → %-16s %-42s → %s\n", step.Tool, in, out)
}

func (o *ConsoleObserver) OnStoryDone(_ int, _ int, story UserStory) {
	dur := story.Duration.Round(time.Millisecond)
	steps := len(story.StepResults)
	switch story.Status {
	case "passed":
		fmt.Fprintf(o.w, "      ✓ passed  (%d steps, %s)\n", steps, dur)
	case "failed":
		fmt.Fprintf(o.w, "      ✗ FAILED: %s\n", story.Error)
	case "skipped":
		fmt.Fprintf(o.w, "      — skipped (%s)\n", story.Error)
	}
}
