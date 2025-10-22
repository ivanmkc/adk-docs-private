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
	"strings"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/model/gemini"
	"google.golang.org/adk/runner"
	"google.golang.org/adk/session"
	"google.golang.org/adk/tool"
	"google.golang.org/genai"
)

const (
	appName   = "customer_support_agent"
	userID    = "user1234"
	sessionID = "1234"
	modelID   = "gemini-2.0-flash"
)

// checkAndTransferArgs defines the arguments for the checkAndTransfer tool.
type checkAndTransferArgs struct {
	Query string `json:"query"`
}

// checkAndTransferResult defines the result of the checkAndTransfer tool.
type checkAndTransferResult struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// checkAndTransfer checks if the query requires escalation and transfers to another agent if needed.
func checkAndTransfer(ctx tool.Context, args checkAndTransferArgs) checkAndTransferResult {
	if strings.Contains(strings.ToLower(args.Query), "urgent") {
		fmt.Println("Tool: Detected urgency, transferring to the support agent.")
		ctx.Actions().TransferToAgent = "support_agent"
		return checkAndTransferResult{
			Status:  "transferring",
			Message: "Transferring to the support agent...",
		}
	}
	return checkAndTransferResult{
		Status:  "processed",
		Message: fmt.Sprintf("Processed query: '%s'. No further action needed.", args.Query),
	}
}

func main() {
	ctx := context.Background()

	// Create Tools
	escalationTool, err := tool.NewFunctionTool[checkAndTransferArgs, checkAndTransferResult](
		tool.FunctionToolConfig{
			Name:        "check_and_transfer",
			Description: "Checks if the query requires escalation and transfers to another agent if needed.",
		},
		checkAndTransfer,
	)
	if err != nil {
		log.Fatalf("Failed to create escalation tool: %v", err)
	}

	// Create Model
	model, err := gemini.NewModel(ctx, modelID, nil)
	if err != nil {
		log.Fatalf("Failed to create model: %v", err)
	}

	// Create Agents
	supportAgent, err := llmagent.New(llmagent.Config{
		Name:  "support_agent",
		Model: model,
		Instruction: `You are the dedicated support agent.
Mentioned you are a support handler and please help the user with their urgent issue.`,
	})
	if err != nil {
		log.Fatalf("Failed to create support agent: %v", err)
	}

	mainAgent, err := llmagent.New(llmagent.Config{
		Name:  "main_agent",
		Model: model,
		Instruction: `You are the first point of contact for customer support of an analytics tool.
Answer general queries.
If the user indicates urgency, use the 'check_and_transfer' tool.`,
		Tools:     []tool.Tool{escalationTool},
	})
	if err != nil {
		log.Fatalf("Failed to create main agent: %v", err)
	}

	// Session and Runner Setup
	sessionService := session.InMemoryService()
	_, err = sessionService.Create(ctx, &session.CreateRequest{
		AppName:   appName,
		UserID:    userID,
		SessionID: sessionID,
	})
	if err != nil {
		log.Fatalf("Failed to create session: %v", err)
	}

	r, err := runner.New(runner.Config{
		AppName:        appName,
		Agent:          agent.Agent(mainAgent),
		SessionService: sessionService,
	})
	if err != nil {
		log.Fatalf("Failed to create runner: %v", err)
	}

	// Agent Interaction
	query := "this is urgent, i cant login"
	fmt.Printf("User Query: %s\n", query)
	content := genai.NewContentFromText(query, "user")

	currentAgent := agent.Agent(mainAgent)

	for {
		var transferTo string
		for event, err := range r.Run(ctx, userID, sessionID, content, &agent.RunConfig{StreamingMode: agent.StreamingModeNone}) {
			if err != nil {
				log.Printf("Agent Error: %v", err)
				continue
			}
			if event.Actions.TransferToAgent != "" {
				transferTo = event.Actions.TransferToAgent
			}
			if event.LLMResponse.Content != nil && len(event.LLMResponse.Content.Parts) > 0 {
				fmt.Printf("Agent Response: %s\n", event.LLMResponse.Content.Parts[0].Text)
			}
		}

		if transferTo == "support_agent" {
			fmt.Println("--- Transferring to support_agent ---")
			currentAgent = supportAgent
			r, err = runner.New(runner.Config{
				AppName:        appName,
				Agent:          currentAgent,
				SessionService: sessionService,
			})
			if err != nil {
				log.Fatalf("Failed to create new runner for support agent: %v", err)
			}
			content = genai.NewContentFromText("The user has an urgent issue, please assist them.", "user")
		} else {
			break
		}
	}
}

