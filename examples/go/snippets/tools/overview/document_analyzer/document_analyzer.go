// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/artifact"
	"google.golang.org/adk/model/gemini"
	"google.golang.org/adk/runner"
	"google.golang.org/adk/session"
	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
	"google.golang.org/genai"
)

// processDocumentArgs holds the arguments for the processDocument tool.
type processDocumentArgs struct {
	DocumentName  string `json:"document_name"`
	AnalysisQuery string `json:"analysis_query"`
}

// processDocumentResult holds the result of the processDocument tool.
type processDocumentResult struct {
	Status           string `json:"status"`
	Message          string `json:"message,omitempty"`
	AnalysisArtifact string `json:"analysis_artifact,omitempty"`
}

// processDocument analyzes a document using context from the tool.Context.
// It demonstrates listing, loading, and saving artifacts according to the agent.Artifacts interface.
func processDocument(toolCtx tool.Context, args processDocumentArgs) processDocumentResult {
	ctx := context.Background()
	// 1. List all available artifacts.
	listResp, err := toolCtx.Artifacts().List(ctx)
	if err != nil {
		errStr := fmt.Sprintf("failed to list artifacts: %v", err)
		fmt.Println("Tool Error: " + errStr)
		return processDocumentResult{Status: "error", Message: errStr}
	}
	fmt.Printf("Tool: Listing all available artifacts: %v\n", listResp.FileNames)

	// 2. Load an artifact.
	fmt.Printf("Tool: Attempting to load artifact: %s\n", args.DocumentName)
	loadResp, err := toolCtx.Artifacts().Load(ctx, args.DocumentName)
	if err != nil {
		// This is a simplified error handling.
		fmt.Printf("Tool: Document '%s' not found: %v\n", args.DocumentName, err)
		return processDocumentResult{
			Status:  "error",
			Message: fmt.Sprintf("Document '%s' not found.", args.DocumentName),
		}
	}

	var documentText string
	if loadResp.Part != nil {
		documentText = loadResp.Part.Text
	}
	fmt.Printf("Tool: Loaded document '%s' (%d chars).\n", args.DocumentName, len(documentText))

	// 3. Perform analysis (placeholder).
	analysisResult := fmt.Sprintf("Analysis of '%s' regarding '%s': The document appears to be about planetary science, discussing the composition of Mars' atmosphere. [Placeholder Analysis Result]", args.DocumentName, args.AnalysisQuery)
	fmt.Println("Tool: Performed analysis.")

	// 4. Save the analysis result as a new artifact.
	analysisPart := genai.NewPartFromText(analysisResult)
	newArtifactName := "analysis_" + args.DocumentName

	_, err = toolCtx.Artifacts().Save(ctx, newArtifactName, analysisPart)
	if err != nil {
		errStr := fmt.Sprintf("failed to save artifact: %v", err)
		fmt.Println("Tool Error: " + errStr)
		return processDocumentResult{Status: "error", Message: errStr}
	}
	fmt.Printf("Tool: Saved analysis artifact as '%s'\n", newArtifactName)

	return processDocumentResult{
		Status:           "success",
		AnalysisArtifact: newArtifactName,
	}
}
func main() {
	ctx := context.Background()

	model, err := gemini.NewModel(ctx, "gemini-2.5-flash", &genai.ClientConfig{
		APIKey: os.Getenv("GOOGLE_API_KEY"),
	})
	if err != nil {
		log.Fatalf("Failed to create model: %v", err)
	}

	processDocumentTool, err := functiontool.New(
		functiontool.Config{
			Name:        "process_document",
			Description: "Analyzes a document, performs a query, and saves the result.",
		},
		processDocument,
	)
	if err != nil {
		log.Fatalf("Failed to create function tool: %v", err)
	}

	docAgent, err := llmagent.New(llmagent.Config{
		Name:        "document_analyzer_agent",
		Model:       model,
		Description: "Agent that can analyze documents provided as artifacts.",
		Instruction: "You are an expert document analyst. When the user asks you to analyze a document, use the process_document tool. You must provide both the document name and a clear analysis query to the tool.",
		Tools:       []tool.Tool{processDocumentTool},
	})
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}

	userID, appName := "test_user", "doc_analyzer_app"
	sessionService := session.InMemoryService()
	// Create session.
	resp, err := sessionService.Create(ctx, &session.CreateRequest{
		AppName: appName,
		UserID:  userID,
	})
	if err != nil {
		log.Fatalf("Failed to create the session service: %v", err)
	}

	session := resp.Session
	artifactService := artifact.InMemoryService()

	// Populate a sample document artifact.
	_, err = artifactService.Save(ctx, &artifact.SaveRequest{
		AppName:   appName,
		UserID:    userID,
		SessionID: session.ID(),
		FileName:  "mars_report.txt",
		Part: genai.NewPartFromText(
			"The atmosphere of Mars is the layer of gases surrounding Mars. " +
				"It is primarily composed of carbon dioxide (95%), molecular nitrogen (2.8%), and argon (2%). " +
				"It also contains trace levels of water vapor, oxygen, carbon monoxide, hydrogen, and other noble gases."),
	})
	if err != nil {
		log.Fatalf("Failed to save artifact: %v", err)
	}

	r, err := runner.New(runner.Config{
		AppName:         appName,
		Agent:           docAgent,
		SessionService:  sessionService,
		ArtifactService: artifactService,
	})
	if err != nil {
		log.Fatalf("Failed to create runner: %v", err)
	}

	fmt.Println("A sample document 'mars_report.txt' has been loaded.")
	fmt.Println("You can ask the agent to analyze it, for example: 'Summarize mars_report.txt for me.'")

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("\nUser -> ")

		userInput, err := reader.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}

		userMsg := genai.NewContentFromText(userInput, genai.RoleUser)

		fmt.Print("\nAgent -> ")
		for event, err := range r.Run(ctx, userID, session.ID(), userMsg, agent.RunConfig{}) {
			if err != nil {
				fmt.Printf("\nAGENT_ERROR: %v\n", err)
			}

			for _, p := range event.LLMResponse.Content.Parts {
				fmt.Print(p.Text)
			}
		}
		fmt.Println()
	}
}
