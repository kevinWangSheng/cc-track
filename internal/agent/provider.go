package agent

// Provider represents an LLM API provider.
type Provider struct {
	Name    string `json:"name"`
	BaseURL string `json:"base_url"`
	APIKey  string `json:"api_key"`
	Model   string `json:"model"`
}

// Built-in providers. API keys are loaded from environment variables.
var builtinProviders = map[string]Provider{
	"zhipu": {
		Name:    "zhipu",
		BaseURL: "https://open.bigmodel.cn/api/anthropic",
		Model:   "GLM-5",
	},
	"minimax": {
		Name:    "minimax",
		BaseURL: "https://api.minimax.chat/anthropic",
		Model:   "MiniMax-M2.7",
	},
}

// DefaultProvider is the default provider name.
const DefaultProvider = "zhipu"

// GetProvider returns a provider by name, with API key from env.
func GetProvider(name string) (Provider, bool) {
	p, ok := builtinProviders[name]
	return p, ok
}

// ListProviders returns all built-in provider names.
func ListProviders() []string {
	var names []string
	for name := range builtinProviders {
		names = append(names, name)
	}
	return names
}
