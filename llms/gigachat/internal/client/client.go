package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	ErrStatusCode = errors.New("unexpected http response code")
)

const (
	RoleAssistant = "assistant"
	RoleUser      = "user"
	RoleFunction  = "function"
	RoleSystem    = "system"
)

type Client struct {
	scope       string
	authData    string
	model       string
	activeToken *TokenResponse
	httpClient  *http.Client
}

func New(scope, authData, model string, cert []byte) (*Client, error) {
	httpClient := makeHttpClient(cert)

	return &Client{
		scope:       scope,
		authData:    authData,
		activeToken: nil,
		httpClient:  httpClient,
		model:       model, // "GigaChat"
	}, nil
}

func (c *Client) GetToken() (*TokenResponse, error) {
	//for safety
	gap := time.Minute

	if c.activeToken == nil || time.Now().Add(gap).Unix() > c.activeToken.ExpiresAt {
		// get new token
		token, err := c.GetNewToken()
		if err != nil {
			return nil, fmt.Errorf("GetNewToken(): %w", err)
		}

		c.activeToken = token
	}

	return c.activeToken, nil
}

func (c *Client) call(_ context.Context, request *CallRequest) (*CallResponse, error) {
	data, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("json.Marshal(requestData): %w", err)
	}

	req, err := http.NewRequest(
		http.MethodPost,
		"https://gigachat.devices.sberbank.ru/api/v1/chat/completions",
		bytes.NewBuffer(data),
	)
	if err != nil {
		return nil, fmt.Errorf("http.NewRequest(): %w", err)
	}

	token, err := c.GetToken()
	if err != nil {
		return nil, fmt.Errorf("c.GetToken(): %w", err)
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Authorization", "Bearer "+token.AccessToken)

	response, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("client.Do(req): %w", err)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("io.ReadAll(response.Body): %w", err)
	}

	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w(%d): %s", ErrStatusCode, response.StatusCode, body)
	}

	var callResponse CallResponse

	err = json.Unmarshal(body, &callResponse)
	if err != nil {
		return nil, fmt.Errorf("json.Unmarshal(body, &result): %w", err)
	}

	return &callResponse, nil
}

func (c *Client) Call(ctx context.Context, prompt string, functions []FunctionDesc) (string, error) {
	requestData := CallRequest{
		Model: c.model,
		Messages: []Message{
			{
				Role:    RoleUser,
				Content: prompt,
			},
		},
		Stream:            false,
		RepetitionPenalty: 1,
		FunctionCall:      "auto",
		Funcions:          functions,
	}

	callResponse, err := c.call(ctx, &requestData)
	if err != nil {
		return "", fmt.Errorf("c.call(ctx, &requestData): %w", err)
	}

	return callResponse.Choices[0].Message.Content, nil
}

type CallRequest struct {
	Model             string         `json:"model"`
	Messages          []Message      `json:"messages"`
	Stream            bool           `json:"stream"`
	RepetitionPenalty int            `json:"repetition_penalty"`
	FunctionCall      string         `jsoh:"function_call"`
	Funcions          []FunctionDesc `jsoh:"functions"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type CallResponse struct {
	Choices []struct {
		FinishReason string  `json:"finish_reason"`
		Index        int     `json:"index"`
		Message      Message `json:"message"`
	} `json:"choices"`
	Created int    `json:"created"`
	Model   string `json:"model"`
	Object  string `json:"object"`
	Usage   struct {
		CompletionTokens int `json:"completion_tokens"`
		PromptTokens     int `json:"prompt_tokens"`
		SystemTokens     int `json:"system_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

func (c *Client) GenerateContent(
	ctx context.Context,
	messages []Message,
	functions []FunctionDesc,
) (*CallResponse, error) {
	requestData := CallRequest{
		Model:             c.model,
		Messages:          messages,
		Stream:            false,
		RepetitionPenalty: 1,
	}
	if functions != nil {
		requestData.FunctionCall = "auto"
		requestData.Funcions = functions
	}

	callResponse, err := c.call(ctx, &requestData)
	if err != nil {
		return nil, fmt.Errorf("c.call(ctx, &requestData): %w", err)
	}

	return callResponse, nil
}

func (c *Client) GetNewToken() (*TokenResponse, error) {
	data := url.Values{}
	data.Set("scope", c.scope)

	req, err := http.NewRequest(
		http.MethodPost,
		"https://ngw.devices.sberbank.ru:9443/api/v2/oauth",
		strings.NewReader(data.Encode()),
	)

	if err != nil {
		return nil, fmt.Errorf("http.NewRequest(): %w", err)
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Authorization", "Basic "+c.authData)
	id := uuid.New()
	req.Header.Add("RqUID", id.String())

	response, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("client.Do(req): %w", err)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("io.ReadAll(response.Body): %w", err)
	}

	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w(%d): %s", ErrStatusCode, response.StatusCode, body)
	}

	var tokenResponse TokenResponse

	err = json.Unmarshal(body, &tokenResponse)
	if err != nil {
		return nil, fmt.Errorf("json.Unmarshal(body, &tokenResponse): %w", err)
	}

	return &tokenResponse, nil
}

type TokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresAt   int64  `json:"expires_at"`
}

type FunctionDesc struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

func makeHttpClient(cert []byte) *http.Client {
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(cert)

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: caCertPool,
			},
		},
	}

	return client
}
