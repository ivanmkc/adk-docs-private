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

	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/model/gemini"
	"google.golang.org/adk/runner"
	"google.golang.org/adk/session"
	"google.golang.org/adk/tool"
	"google.golang.org/genai"
)

// The concept of a "Toolset" for grouping and dynamically providing tools
// is currently a feature specific to the Python ADK. The Go ADK does not have
// a direct `Toolset` equivalent.
//
// The standard and recommended way to provide tools to an agent in Go is to
// create each `tool.Tool` instance individually (e.g., using
// `tool.NewFunctionTool`) and then pass them as a slice (`[]tool.Tool`) to the
// `Tools` field in the `llmagent.Config` struct during agent initialization.
//
// This example demonstrates the functionally equivalent approach in Go:
// 1. Define individual tool functions (`addNumbers`, `subtractNumbers`, `greetUser`).
// 2. Create `FunctionTool` instances for each.
// 3. Collect these tools into a single slice.
// 4. Pass the slice to the agent's configuration.
// This achieves the same result as the Python Toolset example—providing a
// collection of related tools to an agent—using the standard constructs
// available in the Go ADK.

const (
	appName   = "toolset_example_agent"
	userID    = "user1234"
	sessionID = "1234"
	modelID   = "gemini-2.0-flash" // Replace with a valid model name
)

// addNumbersArgs defines the arguments for the addNumbers tool.
type addNumbersArgs struct {
	A int `json:"a"`
	B int `json:"b"`
}

// addNumbersResult defines the result of the addNumbers tool.
type addNumbersResult struct {
	Result int `json:"result"`
}

// addNumbers adds two numbers and stores the result in the session state.
func addNumbers(ctx tool.Context, args addNumbersArgs) addNumbersResult {
	result := args.A + args.B
	ctx.State().Set("last_math_result", result)
	fmt.Printf("Tool: Calculated %d + %d = %d\n", args.A, args.B, result)
	return addNumbersResult{Result: result}
}

// subtractNumbersArgs defines the arguments for the subtractNumbers tool.
type subtractNumbersArgs struct {
	A int `json:"a"`
	B int `json:"b"`
}

// subtractNumbersResult defines the result of the subtractNumbers tool.
type subtractNumbersResult struct {
	Result int `json:"result"`
}

// subtractNumbers subtracts two numbers.
func subtractNumbers(ctx tool.Context, args subtractNumbersArgs) subtractNumbersResult {
	result := args.A - args.B
	fmt.Printf("Tool: Calculated %d - %d = %d\n", args.A, args.B, result)
	return subtractNumbersResult{Result: result}
}

// greetUserArgs defines the arguments for the greetUser tool.
type greetUserArgs struct {
	Name string `json:"name"`
}

// greetUserResult defines the result of the greetUser tool.
type greetUserResult struct {
	Greeting string `json:"greeting"`
}

// greetUser returns a greeting.
func greetUser(ctx tool.Context, args greetUserArgs) greetUserResult {
	return greetUserResult{Greeting: "Hello, " + args.Name}
}

func main() {
	ctx := context.Background()

	// Create Tools
	addTool, err := tool.NewFunctionTool[addNumbersArgs, addNumbersResult](
		tool.FunctionToolConfig{Name: "add_numbers", Description: "Adds two numbers."},
		addNumbers,
	)
	if err != nil {
		log.Fatalf("Failed to create add tool: %v", err)
	}

	subtractTool, err := tool.NewFunctionTool[subtractNumbersArgs, subtractNumbersResult](
		tool.FunctionToolConfig{Name: "subtract_numbers", Description: "Subtracts two numbers."},
		subtractNumbers,
	)
	if err != nil {
		log.Fatalf("Failed to create subtract tool: %v", err)
	}

	greetTool, err := tool.NewFunctionTool[greetUserArgs, greetUserResult](
		tool.FunctionToolConfig{Name: "greet_user", Description: "Greets the user."},
		greetUser,
	)
	if err != nil {
		log.Fatalf("Failed to create greet tool: %v", err)
	}

	// Group tools into a slice, which is the Go equivalent of a Toolset.
	allTools := []tool.Tool{addTool, subtractTool, greetTool}

	// Create Model
	model, err := gemini.NewModel(ctx, modelID, nil)
	if err != nil {
		log.Fatalf("Failed to create model: %v", err)
	}

	// Create Agent
	calculatorAgent, err := llmagent.New(llmagent.Config{
		Name:  "calculator_agent",
		Model: model,
		Instruction: `You are a calculator and greeter.
- Use the 'add_numbers' or 'subtract_numbers' tools for math.
- Use the 'greet_user' tool for greetings.
- After adding, mention the result is stored.`,
		Tools: allTools,
	})
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
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
		Agent:          agent.Agent(calculatorAgent),
		SessionService: sessionService,
	})
	if err != nil {
		log.Fatalf("Failed to create runner: %v", err)
	}

	// Agent Interaction
	query := "What is 15 + 5?"
	fmt.Printf("User Query: %s\n", query)
	content := genai.NewContentFromText(query, "user")

	var events []*session.Event
	for event, err := range r.Run(ctx, userID, sessionID, content, &agent.RunConfig{StreamingMode: agent.StreamingModeNone}) {
		if err != nil {
			log.Printf("Agent Error: %v", err)
			continue
		}
		events = append(events, event)
	}

	if len(events) > 0 {
		lastEvent := events[len(events)-1]
		if lastEvent.LLMResponse.Content != nil && len(lastEvent.LLMResponse.Content.Parts) > 0 {
			fmt.Printf("Agent Response: %s\n", lastEvent.LLMResponse.Content.Parts[0].Text)
		}
	}
}
