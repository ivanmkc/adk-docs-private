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
	"google.golang.org/adk/model"
	"google.golang.org/adk/model/gemini"
	"google.golang.org/adk/runner"
	"google.golang.org/adk/session"
	"google.golang.org/adk/tool"
	"google.golang.org/genai"
)

const (
	appName    = "weather_sentiment_agent"
	userID     = "user1234"
	sessionID  = "1234"
	modelID    = "gemini-2.0-flash"
)

// getWeatherReportArgs defines the arguments for the getWeatherReport tool.
type getWeatherReportArgs struct {
	City string `json:"city"`
}

// getWeatherReportResult defines the result of the getWeatherReport tool.
type getWeatherReportResult struct {
	Status        string `json:"status"`
	Report        string `json:"report,omitempty"`
	ErrorMessage  string `json:"error_message,omitempty"`
}

// getWeatherReport retrieves the current weather report for a specified city.
func getWeatherReport(ctx tool.Context, args getWeatherReportArgs) getWeatherReportResult {
	switch strings.ToLower(args.City) {
	case "london":
		return getWeatherReportResult{
			Status: "success",
			Report: "The current weather in London is cloudy with a temperature of 18 degrees Celsius and a chance of rain.",
		}
	case "paris":
		return getWeatherReportResult{
			Status: "success",
			Report: "The weather in Paris is sunny with a temperature of 25 degrees Celsius.",
		}
	default:
		return getWeatherReportResult{
			Status:       "error",
			ErrorMessage: fmt.Sprintf("Weather information for '%s' is not available.", args.City),
		}
	}
}

// analyzeSentimentArgs defines the arguments for the analyzeSentiment tool.
type analyzeSentimentArgs struct {
	Text string `json:"text"`
}

// analyzeSentimentResult defines the result of the analyzeSentiment tool.
type analyzeSentimentResult struct {
	Sentiment  string  `json:"sentiment"`
	Confidence float64 `json:"confidence"`
}

// analyzeSentiment analyzes the sentiment of the given text.
func analyzeSentiment(ctx tool.Context, args analyzeSentimentArgs) analyzeSentimentResult {
	lowerText := strings.ToLower(args.Text)
	if strings.Contains(lowerText, "good") || strings.Contains(lowerText, "sunny") {
		return analyzeSentimentResult{Sentiment: "positive", Confidence: 0.8}
	}
	if strings.Contains(lowerText, "rain") || strings.Contains(lowerText, "bad") {
		return analyzeSentimentResult{Sentiment: "negative", Confidence: 0.7}
	}
	return analyzeSentimentResult{Sentiment: "neutral", Confidence: 0.6}
}

func main() {
	ctx := context.Background()

	// Create Tools
	weatherTool, err := tool.NewFunctionTool[getWeatherReportArgs, getWeatherReportResult](
		tool.FunctionToolConfig{
			Name:        "get_weather_report",
			Description: "Retrieves the current weather report for a specified city.",
		},
		getWeatherReport,
	)
	if err != nil {
		log.Fatalf("Failed to create weather tool: %v", err)
	}

	sentimentTool, err := tool.NewFunctionTool[analyzeSentimentArgs, analyzeSentimentResult](
		tool.FunctionToolConfig{
			Name:        "analyze_sentiment",
			Description: "Analyzes the sentiment of the given text.",
		},
		analyzeSentiment,
	)
	if err != nil {
		log.Fatalf("Failed to create sentiment tool: %v", err)
	}

	// Create Model
	model, err := gemini.NewModel(ctx, modelID, nil)
	if err != nil {
		log.Fatalf("Failed to create model: %v", err)
	}

	// Create Agent
	weatherSentimentAgent, err := llmagent.New(llmagent.Config{
		Name:  "weather_sentiment_agent",
		Model: model,
		Instruction: `You are a helpful assistant that provides weather information and analyzes the sentiment of user feedback.
**If the user asks about the weather in a specific city, use the 'get_weather_report' tool to retrieve the weather details.**
**If the 'get_weather_report' tool returns a 'success' status, provide the weather report to the user.**
**If the 'get_weather_report' tool returns an 'error' status, inform the user that the weather information for the specified city is not available and ask if they have another city in mind.**
**After providing a weather report, if the user gives feedback on the weather (e.g., 'That's good' or 'I don't like rain'), use the 'analyze_sentiment' tool to understand their sentiment.** Then, briefly acknowledge their sentiment.
You can handle these tasks sequentially if needed.`,
		Tools: []tool.Tool{weatherTool, sentimentTool},
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
		Agent:          agent.Agent(weatherSentimentAgent),
		SessionService: sessionService,
	})
	if err != nil {
		log.Fatalf("Failed to create runner: %v", err)
	}

	// Agent Interaction
	queries := []string{"weather in london?", "That's good."}
	for _, query := range queries {
		fmt.Printf("User Query: %s\n", query)
		content := genai.NewContentFromText(query, "user")

		for event, err := range r.Run(ctx, userID, sessionID, content, &agent.RunConfig{}) {
			if err != nil {
				log.Printf("Agent Error: %v", err)
				continue
			}
			if isFinalResponse(event) {
				texts := textParts(event.LLMResponse.Content)
				if len(texts) > 0 {
					fmt.Printf("Agent Response: %s\n", texts[0])
				}
			}
		}
	}
}

func isFinalResponse(ev *session.Event) bool {
	if (ev.Actions.SkipSummarization) || len(ev.LongRunningToolIDs) > 0 {
		return true
	}
	if ev.LLMResponse == nil {
		return true
	}
	return !hasFunctionCalls(ev.LLMResponse) && !hasFunctionResponses(ev.LLMResponse) && !ev.LLMResponse.Partial && !hasTrailingCodeExecutionResult(ev.LLMResponse)
}

func hasFunctionCalls(resp *model.LLMResponse) bool {
	if resp == nil || resp.Content == nil {
		return false
	}
	for _, part := range resp.Content.Parts {
		if part.FunctionCall != nil {
			return true
		}
	}
	return false
}

func hasFunctionResponses(resp *model.LLMResponse) bool {
	if resp == nil || resp.Content == nil {
		return false
	}
	for _, part := range resp.Content.Parts {
		if part.FunctionResponse != nil {
			return true
		}
	}
	return false
}

func hasTrailingCodeExecutionResult(resp *model.LLMResponse) bool {
	if resp == nil || resp.Content == nil || len(resp.Content.Parts) == 0 {
		return false
	}
	lastPart := resp.Content.Parts[len(resp.Content.Parts)-1]
	return lastPart.CodeExecutionResult != nil
}

func textParts(c *genai.Content) (ret []string) {
	if c == nil {
		return nil
	}
	for _, p := range c.Parts {
		if p.Text != "" {
			ret = append(ret, p.Text)
		}
	}
	return ret
}

