package gigachat

const (
	certFilePathEnvVarName = "GIGACHAT_CERT_PATH"
	authDataEnvVarName     = "GIGACHAT_AUTH_DATA"
	modelEnvVarName        = "GIGACHAT_MODEL" //nolint:gosec
	scopeEnvVarName        = "GIGACHAT_SCOPE" //nolint:gosec
)

type options struct {
	cert     []byte
	model    string
	scope    string
	authData string
	// organization string
	// apiType      APIType
	// httpClient   openaiclient.Doer

	// responseFormat *ResponseFormat

	// // required when APIType is APITypeAzure or APITypeAzureAD
	// apiVersion     string
	// embeddingModel string

	// callbackHandler callbacks.Handler
}

// Option is a functional option for the OpenAI client.
type Option func(*options)
