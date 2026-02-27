package ai

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"
)

const lineWidth = 64

type EndpointInfo struct {
	Method  string
	Path    string
	Handler string
	Comment string
}

type ModelInfo struct {
	Name   string
	Fields []string // "ID int64", "Name string"
}

type ProjectContext struct {
	Endpoints []EndpointInfo
	Models    []ModelInfo
	Tables    []string
	Topics    []string
	Summary   string
}

type StepResult struct {
	Tool   string
	Input  string // JSON string
	Output string // JSON string with result
}

type UserStory struct {
	Title       string
	Description string
	Steps       []string
	StepResults []StepResult
	Status      string // "passed" | "failed" | "skipped"
	Error       string
	Duration    time.Duration
}

type Report struct {
	ProjectSummary string
	UserStories    []UserStory
	TotalSteps     int
	Passed         int
	Failed         int
	Skipped        int
	Duration       time.Duration
}

func buildReport(stories []UserStory, totalSteps int, duration time.Duration) *Report {
	r := &Report{
		UserStories: stories,
		TotalSteps:  totalSteps,
		Duration:    duration,
	}
	for _, s := range stories {
		switch s.Status {
		case "passed":
			r.Passed++
		case "failed":
			r.Failed++
		case "skipped":
			r.Skipped++
		}
	}
	return r
}

func (r *Report) String() string {
	var sb strings.Builder
	hr := strings.Repeat("━", lineWidth)

	// ── Header ──────────────────────────────────────────────
	sb.WriteString(hr + "\n")
	title := " Integration Test Report"
	meta := fmt.Sprintf("%s · %d stories · %d steps",
		r.Duration.Round(time.Millisecond), len(r.UserStories), r.TotalSteps)
	sb.WriteString(fmt.Sprintf("%-*s%s\n", lineWidth-len(meta), title, meta))
	sb.WriteString(hr + "\n")

	// ── Stories ──────────────────────────────────────────────
	for _, s := range r.UserStories {
		steps := len(s.StepResults)
		dur := s.Duration.Round(time.Millisecond)

		switch s.Status {
		case "passed":
			right := fmt.Sprintf("%d steps  %s", steps, dur)
			title := truncate(s.Title, lineWidth-8-len(right))
			sb.WriteString(fmt.Sprintf("  PASS  %-*s  %s\n", lineWidth-8-len(right), title, right))

		case "failed":
			right := fmt.Sprintf("%d steps  %s", steps, dur)
			title := truncate(s.Title, lineWidth-8-len(right))
			sb.WriteString(fmt.Sprintf("  FAIL  %-*s  %s\n", lineWidth-8-len(right), title, right))
			if s.Error != "" {
				sb.WriteString(fmt.Sprintf("        Reason: %s\n", s.Error))
			}
			if len(s.StepResults) > 0 {
				sb.WriteString("        Steps executed:\n")
				for i, sr := range s.StepResults {
					in := truncate(sr.Input, 28)
					out := truncate(sr.Output, 22)
					sb.WriteString(fmt.Sprintf("          [%d] %-18s %-28s → %s\n",
						i+1, sr.Tool, in, out))
				}
			}

		case "skipped":
			title := truncate(s.Title, lineWidth-10)
			sb.WriteString(fmt.Sprintf("  SKIP  %-*s  %s\n", lineWidth-10, title, s.Error))

		default:
			sb.WriteString(fmt.Sprintf("  ????  %s\n", s.Title))
		}
	}

	// ── Summary ──────────────────────────────────────────────
	sb.WriteString(hr + "\n")
	sb.WriteString(fmt.Sprintf("  %d passed   %d failed   %d skipped   │   %d steps   │   %s\n",
		r.Passed, r.Failed, r.Skipped, r.TotalSteps, r.Duration.Round(time.Millisecond)))
	sb.WriteString(hr + "\n")

	return sb.String()
}

func (r *Report) Print(w io.Writer) {
	io.WriteString(w, r.String()) //nolint:errcheck
}

func truncate(s string, max int) string {
	if max <= 3 {
		return s
	}
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

func (r *Report) JSON() []byte {
	b, _ := json.MarshalIndent(r, "", "  ")
	return b
}
