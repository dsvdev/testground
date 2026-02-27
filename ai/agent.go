package ai

import (
	"context"
	"time"
)

// Agent orchestrates project analysis, story generation, and story execution.
type Agent struct {
	cfg      options
	executor *Executor
	tools    []Tool
}

// New creates an Agent with the given options.
func New(opts ...Option) *Agent {
	cfg := defaultOptions()
	for _, o := range opts {
		o(&cfg)
	}
	executor := newExecutor(cfg)
	return &Agent{
		cfg:      cfg,
		executor: executor,
		tools:    AvailableTools(cfg),
	}
}

// Plan analyzes the project and generates user stories without executing them.
func (a *Agent) Plan(ctx context.Context) ([]UserStory, error) {
	project, err := Analyze(a.cfg.projectPath)
	if err != nil {
		return nil, err
	}
	return Generate(ctx, a.cfg.llm, project)
}

// Run executes the given user stories and returns a report.
func (a *Agent) Run(ctx context.Context, stories []UserStory) (*Report, error) {
	start := time.Now()
	var results []UserStory
	totalSteps := 0
	total := len(stories)

	for i, story := range stories {
		if totalSteps >= a.cfg.maxStepsTotal {
			story.Status = "skipped"
			story.Error = "maxStepsTotal reached"
			a.cfg.obs.OnStoryStart(i, total, story)
			a.cfg.obs.OnStoryDone(i, total, story)
			results = append(results, story)
			continue
		}

		r := runStory(ctx, a.cfg.llm, a.executor, a.tools, story, RunConfig{
			MaxStepsPerStory: a.cfg.maxStepsPerStory,
			StepsRemaining:   a.cfg.maxStepsTotal - totalSteps,
		}, a.cfg.obs, i, total)
		totalSteps += r.steps
		results = append(results, r.story)
	}

	return buildReport(results, totalSteps, time.Since(start)), nil
}

// RunAll combines Plan and Run in a single call.
func (a *Agent) RunAll(ctx context.Context) (*Report, error) {
	stories, err := a.Plan(ctx)
	if err != nil {
		return nil, err
	}
	return a.Run(ctx, stories)
}
