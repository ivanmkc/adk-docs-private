package main

import (
	"context"
	"iter"
	"log"
	"strconv"
	"strings"

	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/cmd/launcher/adk"
	"google.golang.org/adk/cmd/launcher/web"
	"google.golang.org/adk/cmd/launcher/web/a2a"
	"google.golang.org/adk/model"
	"google.golang.org/adk/server/restapi/services"
	"google.golang.org/adk/session"
	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
	"google.golang.org/genai"
)

// isPrime checks if a number is prime.
func isPrime(n int) bool {
	if n <= 1 {
		return false
	}
	for i := 2; i*i <= n; i++ {
		if n%i == 0 {
			return false
		}
	}
	return true
}

type checkPrimeToolArgs struct {
	Nums []int `json:"nums"`
}

func checkPrimeTool(tc tool.Context, args checkPrimeToolArgs) map[string]any {
	results := make(map[int]bool)
	for _, num := range args.Nums {
		results[num] = isPrime(num)
	}
	return map[string]any{"results": results}
}

func main() {
	primeTool, err := functiontool.New(functiontool.Config{
		Name:        "prime_checking",
		Description: "Check if numbers in a list are prime using efficient mathematical algorithms",
	}, checkPrimeTool)
	if err != nil {
		log.Fatalf("Failed to create prime_checking tool: %v", err)
	}

	primeAgent, err := llmagent.New(llmagent.Config{
		Name:        "check_prime_agent",
		Description: "An agent specialized in checking whether numbers are prime. It can efficiently determine the primality of individual numbers or lists of numbers.",
		Model:       &dummyLLM{}, // A dummy LLM as we are just exposing a tool.
		Tools:       []tool.Tool{primeTool},
	})
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}

	// Create launcher. The a2a.NewLauncher() will dynamically generate the agent card.
	port := 8001
	launcher := web.NewLauncher(a2a.NewLauncher())
	_, err = launcher.Parse([]string{
		"--port", strconv.Itoa(port),
		"a2a", "--a2a_agent_url", "http://localhost:" + strconv.Itoa(port),
	})
	if err != nil {
		log.Fatalf("launcher.Parse() error = %v", err)
	}

	// Create ADK config
	config := &adk.Config{
		AgentLoader:    services.NewSingleAgentLoader(primeAgent),
		SessionService: session.InMemoryService(),
	}

	log.Printf("Starting A2A prime checker server on port %d\n", port)
	// Run launcher
	if err := launcher.Run(context.Background(), config); err != nil {
		log.Fatalf("launcher.Run() error = %v", err)
	}
}

// dummyLLM is a placeholder as llmagent requires a model.
type dummyLLM struct{}

func (d *dummyLLM) Name() string {
	return "dummy-llm-for-prime-agent"
}

func (d *dummyLLM) GenerateContent(ctx context.Context, req *model.LLMRequest, stream bool) iter.Seq2[*model.LLMResponse, error] {
	return func(yield func(*model.LLMResponse, error) bool) {
		// This LLM will just call the tool if it sees it in the prompt.
		// This is a simplification for the example.
		for _, content := range req.Contents {
			for _, part := range content.Parts {
				if strings.Contains(part.Text, "prime") {
					// A real implementation would parse numbers from the text.
					yield(&model.LLMResponse{
						Content: &genai.Content{
							Parts: []*genai.Part{
								{FunctionCall: &genai.FunctionCall{Name: "prime_checking", Args: map[string]any{"nums": []int{7}}}},
							},
						},
					}, nil)
					return
				}
			}
		}
		yield(&model.LLMResponse{
			Content: &genai.Content{
				Parts: []*genai.Part{genai.NewPartFromText("I can check for prime numbers.")},
			},
		}, nil)
	}
}

func (m *dummyLLM) CountTokens(ctx context.Context, req *model.LLMRequest) (int, error) {
	return 0, nil // Not implemented for mock
}