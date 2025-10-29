package main

import (
	"context"
	"fmt"
	"log"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/runner"
	"google.golang.org/adk/session"

	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/model/gemini"
	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
	"google.golang.org/genai"
)

const (
	userID  = "user1234"
	appName = "Google Search_agent"
)

type retrieveOrderByIdArgs struct {
	OrderID string `json:"order_id"`
}

type retrieveOrderByIdResult struct {
	Status string `json:"status"`
}

func retrieveOrderById(ctx tool.Context, args retrieveOrderByIdArgs) retrieveOrderByIdResult {
	switch args.OrderID {
	case "order_001":
		return retrieveOrderByIdResult{
			Status: "Shipped",
		}
	case "order_002":
		return retrieveOrderByIdResult{
			Status: "Processing",
		}
	}
	return retrieveOrderByIdResult{
		Status: "Complete",
	}
}

// This main function is for compilation purposes and does not run the snippets.
func main() {
	ctx := context.Background()

	model, err := gemini.NewModel(ctx, "gemini-2.5-flash", &genai.ClientConfig{})

	if err != nil {
		log.Fatalf("failed to create model: %v", err)
	}

	// Define a custom tool to retrieve customer orders by ID.
	customerOrderTool, err := functiontool.New(
		functiontool.Config{
			Name:        "retrieveOrderById",
			Description: "Retrieves customer orders by id.",
		}, retrieveOrderById)

	if err != nil {
		log.Fatalf("failed to create customer order tool: %v", err)
	}

	customerAgent, err := llmagent.New(llmagent.Config{
		Name:        "order_status_agent",
		Description: "Helps customers with retrieving their order status using a custom tool.",
		Model:       model,
		Instruction: `You can answer questions about customer orders. When a user asks for the status of an order, use the retrieveOrderById tool.`,
		Tools: []tool.Tool{
			customerOrderTool,
		},
	})

	if err != nil {
		log.Fatalf("failed to create science teacher agent: %v", err)
	}

	// Setting up the runner and session
	sessionService := session.InMemoryService()
	config := runner.Config{
		AppName:        appName,
		Agent:          customerAgent,
		SessionService: sessionService,
	}
	r, err := runner.New(config)

	if err != nil {
		log.Fatalf("failed to create the runner: %v", err)
	}

	session, err := sessionService.Create(ctx, &session.CreateRequest{
		AppName: appName,
		UserID:  userID,
	})

	if err != nil {
		log.Fatalf("failed to create the session service: %v", err)
	}

	sessionID := session.Session.ID()
	prompt := "What's the status for order_002?"

	userMsg := &genai.Content{
		Parts: []*genai.Part{{Text: prompt}},
		Role:  string(genai.RoleUser),
	}

	for event, err := range r.Run(ctx, userID, sessionID, userMsg, agent.RunConfig{
		StreamingMode: agent.StreamingModeNone,
	}) {
		if err != nil {
			fmt.Printf("\nAGENT_ERROR: %v\n", err)
		} else {
			for _, p := range event.Content.Parts {
				fmt.Print(p.Text)
			}
		}
	}
}
