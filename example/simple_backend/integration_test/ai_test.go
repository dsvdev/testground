package integration_test

import (
	"fmt"
	"github.com/dsvdev/testground/ai"
	"testing"
)

func TestAiPlan(t *testing.T) {
	analyze, err := ai.Analyze("../../simple_backend")
	if err != nil {
		t.Fatal(err)
		return
	}

	generate, err := ai.Generate(t.Context(), llmClient, analyze)
	if err != nil {
		return
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
