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
	"context"
	"fmt"
	"log"
	"os"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/cmd/launcher/adk"
	"google.golang.org/adk/cmd/launcher/full"
	"google.golang.org/adk/cmd/restapi/services"
	"google.golang.org/adk/model/gemini"
	"google.golang.org/adk/session"
	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/geminitool"
	"google.golang.org/genai"
)

const (
	modelName = "gemini-2.5-flash"
)

// This example demonstrates how to initialize and use the VertexAiSessionService
// with Vertex AI Express Mode, and how to create and run an Agent Engine.
//
// Before running, ensure you have set up your environment:
//  1. Sign up for Vertex AI Express Mode.
//  2. Create an Agent Engine to get an AGENT_ENGINE_ID.
//  3. Set the following environment variables:
//     export GOOGLE_API_KEY="your-express-mode-api-key"
//     export GOOGLE_CLOUD_PROJECT="your-gcp-project-id"
//     export GOOGLE_CLOUD_LOCATION="your-gcp-location"
func main() {
	ctx := context.Background()

	// --8<-- [start:session_service]
	// The appName should be the Reasoning Engine ID.
	// In a real application, get this from your Agent Engine in the Google Cloud Console.
	agentEngineID := "your-reasoning-engine-id" // <-- Replace with your actual Agent Engine ID

	// When using Vertex AI Express Mode with an API key, project and location
	// are typically picked up from environment variables, and you can initialize
	// the service like this.
	sessionService, err := session.VertexAIService(ctx, agentEngineID)
	if err != nil {
		// This will fail if GOOGLE_API_KEY is not set.
		log.Fatalf("Failed to create Vertex AI session service: %v", err)
	}

	fmt.Println("Successfully initialized VertexAiSessionService.")

	// You can now use the sessionService to manage sessions.
	userID := "express-mode-user-go"
	sessionID := "express-mode-session-go"

	fmt.Printf("Creating session %s...\n", sessionID)
	createResp, err := sessionService.Create(ctx, &session.CreateRequest{
		AppName:   agentEngineID,
		UserID:    userID,
		SessionID: sessionID,
	})
	if err != nil {
		log.Fatalf("Failed to create session: %v", err)
	}

	fmt.Printf("Successfully created session with ID: %s\n", createResp.Session.ID())

	// Session cleanup is not explicitly supported by VertexAIService in Express Mode.
	// Sessions are typically ephemeral and managed automatically.
	// --8<-- [end:session_service]

	// --8<-- [start:agent_engine]
	// Create the root agent for the Agent Engine.
	rootAgent, err := createAgent()
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}

	// Configure the ADK with the session service and the agent.
	config := &adk.Config{
		SessionService: sessionService,
		AgentLoader:    services.NewSingleAgentLoader(rootAgent),
	}

	// Use the full launcher to run the agent engine.
	// This will start a server that can handle requests.
	fmt.Println("Starting Agent Engine...")
	l := full.NewLauncher()
	if err := l.Execute(ctx, config, os.Args[1:]); err != nil {
		log.Fatalf("run failed: %v\n\n%s", err, l.CommandLineSyntax())
	}
	// --8<-- [end:agent_engine]
}

func createAgent() (agent.Agent, error) {
	ctx := context.Background()

	model, err := gemini.NewModel(ctx, modelName, &genai.ClientConfig{})
	if err != nil {
		return nil, fmt.Errorf("failed to create gemini model: %w", err)
	}

	a, err := llmagent.New(llmagent.Config{
		Name:        "example_agent",
		Model:       model,
		Description: "An example agent that can use Google Search.",
		Instruction: "You are a helpful assistant. Use your tools to answer the user's questions.",
		Tools:       []tool.Tool{geminitool.GoogleSearch{}},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create llmagent: %w", err)
	}
	return a, nil
}
