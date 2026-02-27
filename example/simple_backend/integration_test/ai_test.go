package integration_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/dsvdev/testground/ai"
)

func TestAiPlan(t *testing.T) {
	if os.Getenv("ANTHROPIC_API_KEY") == "" {
		t.Skip("ANTHROPIC_API_KEY not set")
	}

	analyze, err := ai.Analyze("../../simple_backend")
	if err != nil {
		t.Fatal(err)
	}

	generate, err := ai.Generate(t.Context(), llmClient, analyze)
	if err != nil {
		t.Fatal("generate:", err)
	}

	for _, story := range generate {
		fmt.Println("STORY:")
		fmt.Println("Title:", story.Title)
		fmt.Println("Description:", story.Description)
		fmt.Println("Steps:")
		for i, step := range story.Steps {
			fmt.Println("\t\t", i+1, "-", step)
		}
		fmt.Println("_______________")

	}

	fmt.Println("Starting execution")

	run, err := aiAgent.Run(t.Context(), generate)
	if err != nil {
		t.Fatal(err.Error())
		return
	}

	fmt.Println(run.String())
}
