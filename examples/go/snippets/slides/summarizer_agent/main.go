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
	"google.golang.org/adk/tool/agenttool"
	"google.golang.org/genai"
)

const (
	userID  = "user1234"
	appName = "summarizer_agent"
)

func agentToolExample(ctx context.Context) {
	model, err := gemini.NewModel(ctx, "gemini-2.5-flash", &genai.ClientConfig{})
	if err != nil {
		log.Fatalf("failed to create model: %v", err)
	}

	// Create the summarizer agent (sub-agent)
	summaryAgent, err := llmagent.New(llmagent.Config{
		Name:        "summary_agent",
		Model:       model,
		Description: "Agent to summarize text.",
		Instruction: `You are an expert summarizer. Please read the following text and provide a concise summary`,
	})
	if err != nil {
		log.Fatalf("failed to create summary agent: %v", err)
	}

	// First, wrap the summaryAgent as a tool
	summaryTool := agenttool.New(summaryAgent, nil)

	// Configure and create the root agent
	rootAgent, err := llmagent.New(llmagent.Config{
		Name:        "root_agent",
		Description: "Main agent",
		Model:       model,
		Instruction: `You are a helpful assistant.
When the user asks to summarize some text,
use the 'summarize' tool to generate a summary.
Always forward the user's message exactly as received to the
'summarize' tool, without modifying or summarizing it yourself.
Present the response from the tool to the user.`,
		Tools: []tool.Tool{summaryTool}, // Include the AgentTool
	})
	if err != nil {
		log.Fatalf("failed to create root agent: %v", err)
	}

	// Setting up the runner and session
	sessionService := session.InMemoryService()
	config := runner.Config{
		AppName:        appName,
		Agent:          rootAgent,
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
	prompt := "Please summarize the following text: Ai-Khanoum (/aɪ ˈhɑːnjuːm/, meaning 'Lady Moon';[2] Uzbek: Oyxonim) is the archaeological site of a Hellenistic city in Takhar Province, Afghanistan. The city, whose original name is unknown,[a] was likely founded by an early ruler of the Seleucid Empire and served as a military and economic centre for the rulers of the Greco-Bactrian Kingdom until its destruction c. 145 BC. Rediscovered in 1961, the ruins of the city were excavated by a French team of archaeologists until the outbreak of conflict in Afghanistan in the late 1970s. The city was probably founded between 300 and 285 BC by an official acting on the orders of Seleucus I Nicator or his son Antiochus I Soter, the first two rulers of the Seleucid dynasty. There is a possibility that the site was known to the earlier Achaemenid Empire, who established a small fort nearby. Ai-Khanoum was originally thought to have been a foundation of Alexander the Great, perhaps as Alexandria Oxiana, but this theory is now considered unlikely. Located at the confluence of the Amu Darya (a.k.a. Oxus) and Kokcha rivers, surrounded by well-irrigated farmland, the city itself was divided between a lower town and a 60-metre-high (200 ft) acropolis. Although not situated on a major trade route, Ai-Khanoum controlled access to both mining in the Hindu Kush and strategically important choke points. Extensive fortifications, which were continually maintained and improved, surrounded the city."

	userMsg := &genai.Content{
		Parts: []*genai.Part{{Text: prompt}},
		Role:  string(genai.RoleUser),
	}

	for event, err := range r.Run(ctx, userID, sessionID, userMsg, agent.RunConfig{
		StreamingMode: agent.StreamingModeNone,
	}) {
		if err != nil {
			fmt.Printf("AGENT_ERROR: %v", err)
		} else {
			for _, p := range event.Content.Parts {
				fmt.Print(p.Text)
			}
		}
	}
}

func subagentExample(ctx context.Context) {
	model, err := gemini.NewModel(ctx, "gemini-2.5-flash", &genai.ClientConfig{})
	if err != nil {
		log.Fatalf("failed to create model: %v", err)
	}

	// Create the summarizer agent (sub-agent)
	summaryAgent, err := llmagent.New(llmagent.Config{
		Name:        "summary_agent",
		Model:       model,
		Description: "Agent to summarize text.",
		Instruction: `You are an expert summarizer. Please read the following text and provide a concise summary`,
	})
	if err != nil {
		log.Fatalf("failed to create summary agent: %v", err)
	}

	// Configure and create the root agent
	rootAgent, err := llmagent.New(llmagent.Config{
		Name:        "root_agent",
		Description: "Main agent",
		Model:       model,
		Instruction: `You are a helpful assistant.
When the user asks to summarize some text,
use the 'summarize' tool to generate a summary.
Always forward the user's message exactly as received to the
'summarize' tool, without modifying or summarizing it yourself.
Present the response from the tool to the user.`,
		SubAgents: []agent.Agent{summaryAgent}, // Include the sub-agent
	})
	if err != nil {
		log.Fatalf("failed to create root agent: %v", err)
	}

	// Setting up the runner and session
	sessionService := session.InMemoryService()
	config := runner.Config{
		AppName:        appName,
		Agent:          rootAgent,
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
	prompt := "Please summarize the following text: Ai-Khanoum (/aɪ ˈhɑːnjuːm/, meaning 'Lady Moon';[2] Uzbek: Oyxonim) is the archaeological site of a Hellenistic city in Takhar Province, Afghanistan. The city, whose original name is unknown,[a] was likely founded by an early ruler of the Seleucid Empire and served as a military and economic centre for the rulers of the Greco-Bactrian Kingdom until its destruction c. 145 BC. Rediscovered in 1961, the ruins of the city were excavated by a French team of archaeologists until the outbreak of conflict in Afghanistan in the late 1970s. The city was probably founded between 300 and 285 BC by an official acting on the orders of Seleucus I Nicator or his son Antiochus I Soter, the first two rulers of the Seleucid dynasty. There is a possibility that the site was known to the earlier Achaemenid Empire, who established a small fort nearby. Ai-Khanoum was originally thought to have been a foundation of Alexander the Great, perhaps as Alexandria Oxiana, but this theory is now considered unlikely. Located at the confluence of the Amu Darya (a.k.a. Oxus) and Kokcha rivers, surrounded by well-irrigated farmland, the city itself was divided between a lower town and a 60-metre-high (200 ft) acropolis. Although not situated on a major trade route, Ai-Khanoum controlled access to both mining in the Hindu Kush and strategically important choke points. Extensive fortifications, which were continually maintained and improved, surrounded the city."

	userMsg := &genai.Content{
		Parts: []*genai.Part{{Text: prompt}},
		Role:  string(genai.RoleUser),
	}

	for event, err := range r.Run(ctx, userID, sessionID, userMsg, agent.RunConfig{
		StreamingMode: agent.StreamingModeNone,
	}) {
		if err != nil {
			fmt.Printf("AGENT_ERROR: %v", err)
		} else {
			for _, p := range event.Content.Parts {
				fmt.Print(p.Text)
			}
		}
	}
}

func main() {
	ctx := context.Background()

	agentToolExample(ctx)
	subagentExample(ctx)
}
