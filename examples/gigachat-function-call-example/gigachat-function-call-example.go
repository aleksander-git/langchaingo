package main

import (
	"context"
	"log"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/gigachat"
)

func main() {
	llm, err := gigachat.New()
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()

	// Start by sending an initial question about the weather to the model, adding
	// "available tools" that include a getCurrentWeather function.
	// Thoroughout this sample, messageHistory collects the conversation history
	// with the model - this context is needed to ensure tool calling works
	// properly.
	messageHistory := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeHuman, "Какая погода в Перми?"),
	}
	respchoice, err := llm.GenerateContent(ctx, messageHistory, llms.WithTools(availableTools))
	if err != nil {
		log.Fatal(err)
	}

	_ = respchoice

}

// availableTools simulates the tools/functions we're making available for
// the model.
var availableTools = []llms.Tool{
	{
		Type: "function",
		Function: &llms.FunctionDefinition{
			Name:        "weather_forecast",
			Description: "Возвращает температуру в указанном местоположении",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"location": map[string]any{
						"type":        "string",
						"description": "Местоположение, например, название города",
					},
				},
				"required": []string{"location"},
			},
		},
	},
}
