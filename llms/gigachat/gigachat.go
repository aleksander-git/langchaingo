package gigachat

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/gigachat/internal/client"
)

type LLM struct {
	client *client.Client
}

// Call implements llms.Model.
func (l *LLM) Call(ctx context.Context, prompt string, options ...llms.CallOption) (string, error) {
	panic("unimplemented")
}

type ChatMessage = client.Message

// GenerateContent implements llms.Model.
func (l *LLM) GenerateContent(
	ctx context.Context,
	messages []llms.MessageContent,
	options ...llms.CallOption,
) (*llms.ContentResponse, error) {

	opts := llms.CallOptions{}
	for _, opt := range options {
		opt(&opts)
	}

	chatMsgs := make([]ChatMessage, 0, len(messages))
	for _, mc := range messages {

		// Look at all the parts in mc; expect to find a single Text part
		var text string
		foundText := false
		for _, p := range mc.Parts {
			switch pt := p.(type) {
			case llms.TextContent:
				if foundText {
					return nil, errors.New("expecting a single Text content")
				}
				foundText = true
				text = pt.Text
			default:
				return nil, errors.New("only support Text parts right now")
			}
		}
		msg := ChatMessage{Content: text}
		switch mc.Role {
		case llms.ChatMessageTypeSystem:
			msg.Role = client.RoleSystem
		case llms.ChatMessageTypeAI:
			msg.Role = client.RoleAssistant
		case llms.ChatMessageTypeHuman:
			msg.Role = client.RoleUser
		case llms.ChatMessageTypeGeneric:
			msg.Role = client.RoleUser
		case llms.ChatMessageTypeFunction:
			msg.Role = client.RoleFunction
		// case llms.ChatMessageTypeTool:
		// 	return nil, fmt.Errorf("role tool is not supported")
		default:
			return nil, fmt.Errorf("role %v is not supported", mc.Role)
		}

		chatMsgs = append(chatMsgs, msg)
	}

	gigaFuncs, err := convertTool(opts.Tools)
	if err != nil {
		return nil, fmt.Errorf("error parsing tools: %w", err)
	}

	result, err := l.client.GenerateContent(ctx, chatMsgs, gigaFuncs)
	if err != nil {
		return nil, fmt.Errorf("error in GenerateContent: %w", err)
	}

	choices := make([]*llms.ContentChoice, len(result.Choices))

	for i, c := range result.Choices {
		choices[i] = &llms.ContentChoice{
			Content:    c.Message.Content,
			StopReason: fmt.Sprint(c.FinishReason),
			GenerationInfo: map[string]any{
				"CompletionTokens": result.Usage.CompletionTokens,
				"PromptTokens":     result.Usage.PromptTokens,
				"TotalTokens":      result.Usage.TotalTokens,
				"ReasoningTokens":  result.Usage.SystemTokens,
			},
		}

	}

	resp := llms.ContentResponse{
		Choices: choices,
	}

	return &resp, nil
}

var _ llms.Model = (*LLM)(nil)

// New returns a new Gigachat LLM.
func New(opts ...Option) (*LLM, error) {
	_, c, err := newClient(opts...)
	if err != nil {
		return nil, err
	}
	return &LLM{
		client: c,
	}, err
}

// newClient creates an instance of the internal client.
func newClient(opts ...Option) (*options, *client.Client, error) {

	caCert, err := os.ReadFile(os.Getenv(certFilePathEnvVarName))
	if err != nil {
		return nil, nil, fmt.Errorf("error reading a certificate: %w", err)
	}

	options := &options{
		scope:    os.Getenv(scopeEnvVarName),
		model:    os.Getenv(modelEnvVarName),
		authData: os.Getenv(authDataEnvVarName),
		cert:     caCert,
	}

	for _, opt := range opts {
		opt(options)
	}

	gigaClient, err := client.New(options.scope, options.authData, options.model, options.cert)
	if err != nil {
		return nil, nil, fmt.Errorf("error creating a clinet: %w", err)
	}

	return options, gigaClient, nil
}

func convertTool(tools []llms.Tool) ([]client.FunctionDesc, error) {
	gigaTools := make([]client.FunctionDesc, 0, len(tools))
	for i, tool := range tools {
		if tool.Type != "function" {
			return nil, fmt.Errorf("tool [%d]: unsupported type %q, want 'function'", i, tool.Type)
		}

		gigaFuncDecl := client.FunctionDesc{
			Name:        tool.Function.Name,
			Description: tool.Function.Description,
		}

		// Expect the Parameters field to be a map[string]any, from which we will
		// extract properties to populate the schema.
		params, ok := tool.Function.Parameters.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("tool [%d]: unsupported type %T of Parameters", i, tool.Function.Parameters)
		}

		_, ok = params["properties"].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("tool [%d]: expected to find a map of properties", i)
		}
		_, ok = params["required"].([]string)
		if !ok {
			return nil, fmt.Errorf("tool [%d]: expected to find a slice of required fields", i)
		}
		_, ok = params["type"].(string)
		if !ok {
			return nil, fmt.Errorf("tool [%d]: expected to find a field type", i)
		}

		gigaFuncDecl.Parameters = params

		gigaTools = append(gigaTools, gigaFuncDecl)

	}

	return gigaTools, nil
}
