package main

import (
	"context"
	"fmt"
	"log"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/artifact"
	"google.golang.org/adk/memory"
	"google.golang.org/adk/model/gemini"
	"google.golang.org/adk/runner"
	"google.golang.org/adk/session"
	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
	"google.golang.org/genai"
)

func main() {
	ctx := context.Background()
	model, err := gemini.NewModel(ctx, "gemini-2.0-flash", &genai.ClientConfig{})
	if err != nil {
		log.Fatal(err)
	}

	docAnalysisTool, err := functiontool.New(
		functiontool.Config{
			Name:        "process_document",
			Description: "Analyzes a document using context from memory.",
		},
		processDocument,
	)
	if err != nil {
		log.Fatal(err)
	}

	mainAgent, err := llmagent.New(llmagent.Config{
		Name:        "main_agent",
		Model:       model,
		Instruction: "You are an agent that can process documents.",
		Tools:       []tool.Tool{docAnalysisTool},
	})
	if err != nil {
		log.Fatal(err)
	}

	sessionService := session.InMemoryService()
	artifactService := artifact.InMemoryService()
	memoryService := memory.InMemoryService()
	runner, err := runner.New(runner.Config{
		AppName:         "doc_analysis",
		Agent:           mainAgent,
		SessionService:  sessionService,
		ArtifactService: artifactService,
		MemoryService:   memoryService,
	})
	if err != nil {
		log.Fatal(err)
	}

	session, err := sessionService.Create(ctx, &session.CreateRequest{
		AppName: "doc_analysis",
		UserID:  "user1234",
	})
	if err != nil {
		log.Fatal(err)
	}

	// First run to populate memory. The agent will respond, and the runner will
	// automatically add the interaction to the memory service.
	run(ctx, runner, session.Session.ID(), "I am very interested in positive sentiment analysis.")
	// Second run that uses the tool to search the memory populated by the first run.
	run(ctx, runner, session.Session.ID(), "process the document named 'my_document' and analyze it for 'sentiment'")
}

func run(ctx context.Context, r *runner.Runner, sessionID string, prompt string) {
	fmt.Printf("\n> %s\n", prompt)
	events := r.Run(
		ctx,
		"user1234",
		sessionID,
		genai.NewContentFromText(prompt, genai.RoleUser),
		agent.RunConfig{
			StreamingMode: agent.StreamingModeNone,
		},
	)
	for event, err := range events {
		if err != nil {
			log.Fatalf("ERROR during agent execution: %v", err)
		}

		fmt.Printf("Agent Response: %s\n", event.Content.Parts[0].Text)
	}
}
