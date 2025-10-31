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
	"google.golang.org/adk/tool/functiontool"
	"google.golang.org/genai"
)

type lookupOrderStatusArgs struct {
	OrderID string `json:"order_id"`
}

type order struct {
	State          string `json:"state"`
	TrackingNumber string `json:"tracking_number"`
}

type lookupOrderStatusResult struct {
	Status       string `json:"status"`
	Order        order  `json:"order,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
}

func lookupOrderStatus(ctx tool.Context, args lookupOrderStatusArgs) lookupOrderStatusResult {
	// ... function implementation to fetch status ...
	if statusDetails, ok := fetchStatusFromBackend(args.OrderID); ok {
		return lookupOrderStatusResult{
			Status: "success",
			Order: order{
				State:          statusDetails.State,
				TrackingNumber: statusDetails.Tracking,
			},
		}
	}
	return lookupOrderStatusResult{Status: "error", ErrorMessage: fmt.Sprintf("Order ID %s not found.", args.OrderID)}
}

type statusDetails struct {
	State    string
	Tracking string
}

func fetchStatusFromBackend(orderID string) (statusDetails, bool) {
	if orderID == "12345" {
		return statusDetails{State: "shipped", Tracking: "1Z9..."}, true
	}
	return statusDetails{}, false
}

func main() {
	ctx := context.Background()
	model, err := gemini.NewModel(ctx, "gemini-2.0-flash", &genai.ClientConfig{})
	if err != nil {
		log.Fatal(err)
	}

	orderStatusTool, err := functiontool.New(
		functiontool.Config{
			Name:        "lookup_order_status",
			Description: "Fetches the current status of a customer's order using its ID.",
		},
		lookupOrderStatus,
	)
	if err != nil {
		log.Fatal(err)
	}

	mainAgent, err := llmagent.New(llmagent.Config{
		Name:        "main_agent",
		Model:       model,
		Instruction: "You are an agent that can lookup order status.",
		Tools:       []tool.Tool{orderStatusTool},
	})
	if err != nil {
		log.Fatal(err)
	}

	sessionService := session.InMemoryService()
	runner, err := runner.New(runner.Config{
		AppName:        "order_status",
		Agent:          mainAgent,
		SessionService: sessionService,
	})
	if err != nil {
		log.Fatal(err)
	}

	session, err := sessionService.Create(ctx, &session.CreateRequest{
		AppName: "order_status",
		UserID:  "user1234",
	})
	if err != nil {
		log.Fatal(err)
	}

	run(ctx, runner, session.Session.ID(), "what is the status of order 12345?")
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
