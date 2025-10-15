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

	"google.golang.org/adk/session"
)

// This example demonstrates how to initialize and use the VertexAiSessionService
// with Vertex AI Express Mode.
//
// Before running, ensure you have set up your environment:
// 1. Sign up for Vertex AI Express Mode.
// 2. Create an Agent Engine to get an AGENT_ENGINE_ID.
// 3. Set the following environment variables:
//    export GOOGLE_API_KEY="your-express-mode-api-key"
//    export GOOGLE_CLOUD_PROJECT="your-gcp-project-id"
//    export GOOGLE_CLOUD_LOCATION="your-gcp-location"

// --8<-- [start:session_service]
func main() {
	ctx := context.Background()

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

	// Clean up the created session.
	fmt.Printf("Deleting session %s...\n", sessionID)
	err = sessionService.Delete(ctx, &session.DeleteRequest{
		AppName:   agentEngineID,
		UserID:    userID,
		SessionID: sessionID,
	})
	if err != nil {
		log.Fatalf("Failed to delete session: %v", err)
	}
	fmt.Println("Successfully deleted session.")
}

