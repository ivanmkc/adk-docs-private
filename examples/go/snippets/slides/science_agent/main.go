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
	"google.golang.org/adk/tool/geminitool"
	"google.golang.org/genai"
)

const (
	userID  = "user1234"
	appName = "Google Search_agent"
)

// This main function is for compilation purposes and does not run the snippets.
func main() {
	ctx := context.Background()

	model, err := gemini.NewModel(ctx, "gemini-2.5-flash", &genai.ClientConfig{})

	if err != nil {
		log.Fatalf("failed to create model: %v", err)
	}

	// Create a Google Search tool instance.
	searchTool := geminitool.GoogleSearch{}

	scienceTeacherAgent, err := llmagent.New(llmagent.Config{
		Name:        "science-app",
		Description: "Science teacher agent",
		Model:       model,
		Instruction: `You are a helpful science teacher that explains science concepts to kids and teenagers.`,
		Tools: []tool.Tool{
			searchTool,
		},
	})

	if err != nil {
		log.Fatalf("failed to create science teacher agent: %v", err)
	}

	// Setting up the runner and session
	sessionService := session.InMemoryService()
	config := runner.Config{
		AppName:        appName,
		Agent:          scienceTeacherAgent,
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
	prompt := "Why is the sky blue?"

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
	//    for {
	//        fmt.Print("\nUser -> ")
	//        userInput, _ := reader.ReadString('\n')
	//        if strings.EqualFold(strings.TrimSpace(userInput), "quit") {
	//            break
	//        }

	// 	          userMsg := genai.NewContentFromText(userInput, genai.RoleUser)

	//        fmt.Print("\nAgent -> ")
	//        for event, _ := range r.Run(ctx, userID, session.Session.ID(), userMsg, agent.RunConfig{
	//            StreamingMode: agent.StreamingModeNone,
	//        }) {
	// 			for _, p := range event.LLMResponse.Content.Parts {
	//                    fmt.Print(p.Text)
	//                }

	//        }

	//    }

}
