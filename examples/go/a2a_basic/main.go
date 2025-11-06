package main

import (
	"bufio"
	"context"
	"fmt"
	"iter"
	"log"
	"os"

	"strings"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/agent/remoteagent"
	"google.golang.org/adk/artifact"
	"google.golang.org/adk/model"
	"google.golang.org/adk/model/gemini"
	"google.golang.org/adk/runner"
	"google.golang.org/adk/session"
	"google.golang.org/adk/tool"

	"google.golang.org/genai"
)

// --- Local Roll Agent ---

// --- Remote Prime Agent ---

// newPrimeAgent creates a new remote A2A agent for checking prime numbers.
func newPrimeAgent() (agent.Agent, error) {
	// Create the remote A2A agent by providing the source of the agent card.
	// The ADK will fetch the agent card from the well-known location at this URL.
	remoteAgent, err := remoteagent.New(remoteagent.A2AConfig{
		Name:            "prime_agent",
		AgentCardSource: "http://localhost:8001",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create remote prime agent: %w", err)
	}
	return remoteAgent, nil
}

// --- Root Agent ---

// newRootAgent creates the main orchestrator agent.
func newRootAgent(ctx context.Context, primeAgent agent.Agent) (agent.Agent, error) {



	model, _ := gemini.NewModel(ctx, "gemini-2.0-flash", &genai.ClientConfig{})
	llmAgent, err := llmagent.New(llmagent.Config{
		Name:        "root_agent",
		Description: "Agent to roll dice and check if numbers are prime.",
		SubAgents:   []agent.Agent{primeAgent},
		Model:       model,
		// Instruction: `You are a helpful assistant that can roll dice and check if numbers are prime.
		// 	You delegate rolling prime checking tasks to the prime_agent.
		// 	Follow these steps:
		// 	1. If the user asks to roll a die, use the roll_dice tool.
		// 	2. If the user asks to check primes, delegate to the prime_agent.
		// 	3. If the user asks to roll a die and then check if the result is prime, use roll_dice tool first, then pass the result to prime_agent.`,
		Tools:       []tool.Tool{},
	})
	if err != nil {
		return nil, err
	}
	return llmAgent, nil
}

// --- Mock LLM ---

type mockLLM struct{}

func newMockLLM() model.LLM {
	return &mockLLM{}
}

func (m *mockLLM) Name() string {
	return "mock-llm"
}

func (m *mockLLM) GenerateContent(ctx context.Context, req *model.LLMRequest, stream bool) iter.Seq2[*model.LLMResponse, error] {
	return func(yield func(*model.LLMResponse, error) bool) {
		textInput := ""
		if len(req.Contents) > 0 {
			lastContent := req.Contents[len(req.Contents)-1]
			if lastContent != nil {
				for _, part := range lastContent.Parts {
					if part.Text != "" {
						textInput = part.Text
						break
					}
				}
			}
		}

		var response *model.LLMResponse

		if strings.Contains(strings.ToLower(textInput), "prime") {
			// Simulate calling prime_checking skill on remote agent
			response = &model.LLMResponse{
				Content: &genai.Content{
					Parts: []*genai.Part{
						{FunctionCall: &genai.FunctionCall{Name: "prime_checking", Args: map[string]any{"nums": []int{7}}}},
					},
				},
			}
		} else {
			response = &model.LLMResponse{
				Content: &genai.Content{
					Parts: []*genai.Part{
						genai.NewPartFromText("I can roll dice or check prime numbers. What would you like?"),
					},
				},
			}
		}
		yield(response, nil)
	}
}

func (m *mockLLM) CountTokens(ctx context.Context, req *model.LLMRequest) (int, error) {
	return 0, nil // Not implemented for mock
}

// --- Main Function ---

func main() {
	ctx := context.Background()
	


	// Initialize remote prime agent
	primeAgent, err := newPrimeAgent()
	if err != nil {
		log.Fatalf("Failed to create prime agent: %v", err)
	}

	// Initialize root agent
	rootAgent, err := newRootAgent(ctx, primeAgent)
	if err != nil {
		log.Fatalf("Failed to create root agent: %v", err)
	}

	// Create a session for the interaction
	sessionService := session.InMemoryService()
	artifactService := artifact.InMemoryService()

	_, err = sessionService.Create(ctx, &session.CreateRequest{
		AppName:   rootAgent.Name(),
		UserID:    "user-123",
		SessionID: "session-abc",
	})
	if err != nil {
		log.Fatalf("Failed to create session: %v", err)
	}

	// Create a runner to execute the root agent
	runnerConfig := runner.Config{
		AppName:         rootAgent.Name(),
		Agent:           rootAgent,
		SessionService:  sessionService,
		ArtifactService: artifactService,
	}
	runner, err := runner.New(runnerConfig)
	if err != nil {
		log.Fatalf("Failed to create runner: %v", err)
	}

	// // Simulate user input
	// userInput := "check if 7 is prime"
	// fmt.Printf("User: %s\n", userInput)

	// // Run the agent with user input
	// inputContent := &genai.Content{Parts: []*genai.Part{genai.NewPartFromText(userInput)}}
	// for event, err := range runner.Run(ctx, "user-123", "session-abc", inputContent, agent.RunConfig{
	// 	StreamingMode: agent.StreamingModeNone,
	// }) {
	// 	if err != nil {
	// 		log.Printf("Agent run error: %v", err)
	// 		continue
	// 	}
	// 	if event.Content != nil {
	// 		for _, part := range event.Content.Parts {
	// 			if part.Text != "" {
	// 				fmt.Printf("Bot: %s\n", part.Text)
	// 			}
	// 			if part.FunctionCall != nil {
	// 				fmt.Printf("Bot calls tool: %s with args: %v\n", part.FunctionCall.Name, part.FunctionCall.Args)
	// 			}
	// 		}
	// 	}
	// }

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("\nUser -> ")

		userInput, err := reader.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}

		userMsg := genai.NewContentFromText(userInput, genai.RoleUser)

		streamingMode := agent.StreamingModeNone

		fmt.Print("\nAgent -> ")
		for event, err := range runner.Run(ctx, "user-123", "session-abc", userMsg, agent.RunConfig{
			StreamingMode: streamingMode,
		}) {
			if err != nil {
				fmt.Printf("\nAGENT_ERROR: %v\n", err)
			} else if event != nil {
				for _, p := range event.Content.Parts {
					// if its running in streaming mode, don't print the non partial llmResponses
					fmt.Print(p.Text)
				}
			}
		}
	}
}
