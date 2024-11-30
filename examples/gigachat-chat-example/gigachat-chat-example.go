package main

import (
	"context"
	"fmt"
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

	content := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, "You are a company branding design wizard."),
		llms.TextParts(llms.ChatMessageTypeHuman, "What would be a good company name a company that makes colorful socks?"),
	}

	resp, err := llm.GenerateContent(ctx, content,
		llms.WithMaxTokens(1024),
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(resp.Choices[0].Content)
}
