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

type processDocumentArgs struct {
	DocumentName  string `json:"document_name"`
	AnalysisQuery string `json:"analysis_query"`
}

type processDocumentResult struct {
	Status           string `json:"status"`
	AnalysisArtifact string `json:"analysis_artifact,omitempty"`
	Version          int64  `json:"version,omitempty"`
	Message          string `json:"message,omitempty"`
}

func processDocument(ctx tool.Context, args processDocumentArgs) processDocumentResult {
	fmt.Printf("Tool: Attempting to load artifact: %s\n", args.DocumentName)
	documentPart, err := ctx.Artifacts().Load(ctx, args.DocumentName)
	if err != nil {
		return processDocumentResult{Status: "error", Message: fmt.Sprintf("Document '%s' not found.", args.DocumentName)}
	}

	documentText := documentPart.Part.Text
	fmt.Printf("Tool: Loaded document '%s' (%d chars).\n", args.DocumentName, len(documentText))

	// 3. Search memory for related context
	fmt.Printf("Tool: Searching memory for context related to: '%s'\n", args.AnalysisQuery)
	memoryResp, err := ctx.SearchMemory(ctx, args.AnalysisQuery)
	if err != nil {
		fmt.Printf("Tool: Error searching memory: %v\n", err)
	}
	fmt.Printf("Tool: Found %d memory results.\n", len(memoryResp.Memories))

	analysisResult := fmt.Sprintf("Analysis of '%s' regarding '%s' using memory context: [Placeholder Analysis Result]", args.DocumentName, args.AnalysisQuery)
	fmt.Println("Tool: Performed analysis.")

	analysisPart := genai.NewPartFromText(analysisResult)
	newArtifactName := fmt.Sprintf("analysis_%s", args.DocumentName)
	version, err := ctx.Artifacts().Save(ctx, newArtifactName, analysisPart)
	if err != nil {
		return processDocumentResult{Status: "error", Message: "Failed to save artifact."}
	}
	fmt.Printf("Tool: Saved analysis result as '%s' version %d.\n", newArtifactName, version.Version)

	return processDocumentResult{
		Status:           "success",
		AnalysisArtifact: newArtifactName,
		Version:          version.Version,
	}
}

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
