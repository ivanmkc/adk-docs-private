package main

import (
	"context"
	"fmt"
	"log"

	"google.golang.org/adk/artifact"
	"google.golang.org/adk/cmd/launcher/adk"
	"google.golang.org/adk/cmd/launcher/web"
	"google.golang.org/adk/cmd/restapi/services"

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

func scienceAgentExample(ctx context.Context) {
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
		Instruction: `You are a helpful science teacher that explains science concepts to students. Reply with under 5 sentences only.`,
		Tools: []tool.Tool{
			searchTool,
		},
	})

	if err != nil {
		log.Fatalf("failed to create science teacher agent: %v", err)
	}

	// Setting up the runner and session
	sessionService := session.InMemoryService()
	// config := runner.Config{
	// 	AppName:        appName,
	// 	Agent:          scienceTeacherAgent,
	// 	SessionService: sessionService,
	// }
	// r, err := runner.New(config)

	// if err != nil {
	// 	log.Fatalf("failed to create the runner: %v", err)
	// }

	// session, err := sessionService.Create(ctx, &session.CreateRequest{
	// 	AppName: appName,
	// 	UserID:  userID,
	// })

	// if err != nil {
	// 	log.Fatalf("failed to create the session service: %v", err)
	// }

	artifactservice := artifact.InMemoryService()

	agentLoader := services.NewStaticAgentLoader(
		scienceTeacherAgent,
		map[string]agent.Agent{
			"science_agent": scienceTeacherAgent,
		},
	)

	// Web UI
	webConfig, _, _ := web.ParseArgs([]string{})
	fmt.Println(webConfig)
	web.Serve(webConfig, &adk.Config{
		SessionService:  sessionService,
		AgentLoader:     agentLoader,
		ArtifactService: artifactservice,
	})


	// // INTERACTIVE LOOP
	// reader := bufio.NewReader(os.Stdin)

	// for {
	// 	fmt.Print("\nUser -> ")

	// 	userInput, err := reader.ReadString('\n')
	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}

	// 	userMsg := genai.NewContentFromText(userInput, genai.RoleUser)

	// 	streamingMode := agent.StreamingModeSSE

	// 	fmt.Print("\nAgent -> ")
	// 	for event, err := range r.Run(ctx, userID, session.Session.ID(), userMsg, agent.RunConfig{
	// 		StreamingMode: streamingMode,
	// 	}) {
	// 		if err != nil {
	// 			fmt.Printf("\nAGENT_ERROR: %v\n", err)
	// 		} else if event != nil {
	// 			for _, p := range event.Content.Parts {
	// 				// if its running in streaming mode, don't print the non partial llmResponses
	// 				if streamingMode != agent.StreamingModeSSE || event.LLMResponse.Partial {
	// 					fmt.Print(p.Text)
	// 				}
	// 			}
	// 		}
	// 	}
	// }

	// SINGLE RUN EXAMPLE
	// sessionID := session.Session.ID()
	// prompt := "Why is the sky blue?"

	// userMsg := &genai.Content{
	// 	Parts: []*genai.Part{{Text: prompt}},
	// 	Role:  string(genai.RoleUser),
	// }

	// for event, err := range r.Run(ctx, userID, sessionID, userMsg, agent.RunConfig{
	// 	StreamingMode: agent.StreamingModeNone,
	// }) {
	// 	if err != nil {
	// 		fmt.Printf("\nAGENT_ERROR: %v\n", err)
	// 	} else {
	// 		for _, p := range event.Content.Parts {
	// 			fmt.Print(p.Text)
	// 		}
	// 	}
	// }
}

func genericAgentExample(ctx context.Context) {
	model, err := gemini.NewModel(ctx, "gemini-2.5-flash", &genai.ClientConfig{})

	if err != nil {
		log.Fatalf("failed to create model: %v", err)
	}

	// Create a Google Search tool instance.
	searchTool := geminitool.GoogleSearch{}

	teacherAgent, err := llmagent.New(llmagent.Config{
		Name:        "teacher-app",
		Description: "Teacher agent",
		Model:       model,
		Instruction: `You are a helpful {topic} teacher that explains {topic} concepts to {audience}`,
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
		Agent:          teacherAgent,
		SessionService: sessionService,
	}
	r, err := runner.New(config)

	if err != nil {
		log.Fatalf("failed to create the runner: %v", err)
	}

	session, err := sessionService.Create(ctx, &session.CreateRequest{
		AppName: appName,
		UserID:  userID,
		State:  map[string]any{
			"topic": "culinary", 
			"audience": "People that only know 4-letter words",
		},
	})

	if err != nil {
		log.Fatalf("failed to create the session service: %v", err)
	}

	sessionID := session.Session.ID()
	prompt := "Why add salt when boiling water for pasta?"

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

func main() {
	ctx := context.Background()

	scienceAgentExample(ctx)
	// genericAgentExample(ctx)
}